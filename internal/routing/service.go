package routing

import (
	"container/heap"
	"fmt"
	"math"
	"strings"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
	"github.com/aressim/internal/geo"
	"github.com/aressim/internal/sim"
)

type Purpose string

const (
	PurposeMove         Purpose = "move"
	PurposeStrike       Purpose = "strike"
	PurposeDefensiveAir Purpose = "defensive_air"
	defaultMarginCells          = 16
	maritimeMarginCells         = 48
	minGridSize                 = 8
)

type Request struct {
	OwnerCountry      string
	Domain            enginev1.UnitDomain
	Purpose           Purpose
	Start             geo.Point
	End               geo.Point
	RelationshipRules sim.RelationshipRules
	CountryCoalitions map[string]string
}

type Result struct {
	Blocked bool
	Reason  string
	Country string
	Points  []geo.Point
}

type pointContext struct {
	geoContext geo.GeoContext
	isLand     bool
}

type routeContext struct {
	req        Request
	pointCache map[string]pointContext
}

type grid struct {
	minLat float64
	minLon float64
	step   float64
	rows   int
	cols   int
}

type node struct {
	row int
	col int
}

type pqItem struct {
	node     node
	priority float64
	index    int
}

type priorityQueue []*pqItem

func (pq priorityQueue) Len() int           { return len(pq) }
func (pq priorityQueue) Less(i, j int) bool { return pq[i].priority < pq[j].priority }
func (pq priorityQueue) Swap(i, j int)      { pq[i], pq[j] = pq[j], pq[i]; pq[i].index = i; pq[j].index = j }
func (pq *priorityQueue) Push(x any) {
	item := x.(*pqItem)
	item.index = len(*pq)
	*pq = append(*pq, item)
}
func (pq *priorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*pq = old[:n-1]
	return item
}

func BuildRoute(req Request) Result {
	req.OwnerCountry = sim.CountryDisplayCode(req.OwnerCountry)
	if req.OwnerCountry == "" {
		return Result{Blocked: true, Reason: "routing owner country is required"}
	}
	if req.Domain == enginev1.UnitDomain_DOMAIN_UNSPECIFIED {
		return Result{Blocked: true, Reason: "routing domain is required"}
	}
	ctx := &routeContext{
		req:        req,
		pointCache: make(map[string]pointContext, 512),
	}
	if !ctx.pointPassable(req.End) {
		reason, country := ctx.explainBlockedPoint(req.End)
		return Result{Blocked: true, Reason: reason, Country: country}
	}
	if ctx.segmentPassable(req.Start, req.End) {
		return Result{Points: []geo.Point{req.End}}
	}

	g := buildGrid(req.Start, req.End, resolutionFor(req.Domain), marginCellsFor(req.Domain))
	startNode := g.closestNode(req.Start)
	endNode := g.closestNode(req.End)
	if !ctx.pointPassable(g.point(startNode)) {
		startNode = g.nearestPassable(ctx, startNode)
	}
	if !ctx.pointPassable(g.point(endNode)) {
		endNode = g.nearestPassable(ctx, endNode)
	}
	if startNode.row < 0 || endNode.row < 0 {
		reason, country := ctx.explainBlockedPoint(req.End)
		return Result{Blocked: true, Reason: reason, Country: country}
	}

	cameFrom := map[node]node{}
	gScore := map[node]float64{startNode: 0}
	openSet := &priorityQueue{}
	heap.Init(openSet)
	heap.Push(openSet, &pqItem{node: startNode, priority: heuristic(g.point(startNode), g.point(endNode))})
	inOpen := map[node]bool{startNode: true}

	for openSet.Len() > 0 {
		current := heap.Pop(openSet).(*pqItem).node
		inOpen[current] = false
		if current == endNode {
			path := reconstructPath(cameFrom, current)
			points := smoothPath(ctx, g, path)
			return Result{Points: append(points, req.End)}
		}
		currentPoint := g.point(current)
		for _, next := range g.neighbors(current) {
			nextPoint := g.point(next)
			if !ctx.pointPassable(nextPoint) {
				continue
			}
			if !ctx.segmentPassable(currentPoint, nextPoint) {
				continue
			}
			tentative := gScore[current] + ctx.traversalCost(currentPoint, nextPoint)
			if best, ok := gScore[next]; ok && tentative >= best {
				continue
			}
			cameFrom[next] = current
			gScore[next] = tentative
			priority := tentative + heuristic(nextPoint, g.point(endNode))
			if !inOpen[next] {
				heap.Push(openSet, &pqItem{node: next, priority: priority})
				inOpen[next] = true
			}
		}
	}

	reason, country := ctx.explainBlockedPoint(req.End)
	if strings.TrimSpace(reason) == "" {
		reason = fmt.Sprintf("%s could not find a legal route", req.OwnerCountry)
	}
	return Result{Blocked: true, Reason: reason, Country: country}
}

func resolutionFor(domain enginev1.UnitDomain) float64 {
	switch domain {
	case enginev1.UnitDomain_DOMAIN_AIR:
		return 0.20
	case enginev1.UnitDomain_DOMAIN_LAND:
		return 0.10
	case enginev1.UnitDomain_DOMAIN_SEA, enginev1.UnitDomain_DOMAIN_SUBSURFACE:
		return 0.04
	default:
		return 0.10
	}
}

func marginCellsFor(domain enginev1.UnitDomain) int {
	switch domain {
	case enginev1.UnitDomain_DOMAIN_SEA, enginev1.UnitDomain_DOMAIN_SUBSURFACE:
		return maritimeMarginCells
	default:
		return defaultMarginCells
	}
}

func buildGrid(start, end geo.Point, step float64, marginCells int) grid {
	minLat := math.Min(start.Lat, end.Lat)
	maxLat := math.Max(start.Lat, end.Lat)
	minLon := math.Min(start.Lon, end.Lon)
	maxLon := math.Max(start.Lon, end.Lon)
	minLat -= step * float64(marginCells)
	maxLat += step * float64(marginCells)
	minLon -= step * float64(marginCells)
	maxLon += step * float64(marginCells)
	rows := max(minGridSize, int(math.Ceil((maxLat-minLat)/step))+1)
	cols := max(minGridSize, int(math.Ceil((maxLon-minLon)/step))+1)
	return grid{
		minLat: minLat,
		minLon: minLon,
		step:   step,
		rows:   rows,
		cols:   cols,
	}
}

func (g grid) point(n node) geo.Point {
	return geo.Point{
		Lat: g.minLat + float64(n.row)*g.step,
		Lon: g.minLon + float64(n.col)*g.step,
	}
}

func (g grid) closestNode(p geo.Point) node {
	row := int(math.Round((p.Lat - g.minLat) / g.step))
	col := int(math.Round((p.Lon - g.minLon) / g.step))
	if row < 0 {
		row = 0
	}
	if col < 0 {
		col = 0
	}
	if row >= g.rows {
		row = g.rows - 1
	}
	if col >= g.cols {
		col = g.cols - 1
	}
	return node{row: row, col: col}
}

func (g grid) neighbors(n node) []node {
	out := make([]node, 0, 8)
	for dr := -1; dr <= 1; dr++ {
		for dc := -1; dc <= 1; dc++ {
			if dr == 0 && dc == 0 {
				continue
			}
			row := n.row + dr
			col := n.col + dc
			if row < 0 || row >= g.rows || col < 0 || col >= g.cols {
				continue
			}
			out = append(out, node{row: row, col: col})
		}
	}
	return out
}

func (g grid) nearestPassable(ctx *routeContext, start node) node {
	if ctx.pointPassable(g.point(start)) {
		return start
	}
	best := node{row: -1, col: -1}
	bestDist := math.MaxFloat64
	maxRadius := max(g.rows, g.cols)
	for radius := 1; radius < maxRadius; radius++ {
		for row := max(0, start.row-radius); row <= min(g.rows-1, start.row+radius); row++ {
			for col := max(0, start.col-radius); col <= min(g.cols-1, start.col+radius); col++ {
				if absInt(row-start.row) != radius && absInt(col-start.col) != radius {
					continue
				}
				candidate := node{row: row, col: col}
				if !ctx.pointPassable(g.point(candidate)) {
					continue
				}
				d := heuristic(g.point(start), g.point(candidate))
				if d < bestDist {
					bestDist = d
					best = candidate
				}
			}
		}
		if best.row >= 0 {
			return best
		}
	}
	return best
}

func reconstructPath(cameFrom map[node]node, current node) []node {
	path := []node{current}
	for {
		prev, ok := cameFrom[current]
		if !ok {
			break
		}
		path = append(path, prev)
		current = prev
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

func smoothPath(ctx *routeContext, g grid, path []node) []geo.Point {
	if len(path) <= 1 {
		return nil
	}
	points := make([]geo.Point, 0, len(path))
	currentIdx := 0
	for currentIdx < len(path)-1 {
		nextIdx := len(path) - 1
		for nextIdx > currentIdx+1 {
			if ctx.segmentPassable(g.point(path[currentIdx]), g.point(path[nextIdx])) {
				break
			}
			nextIdx--
		}
		points = append(points, g.point(path[nextIdx]))
		currentIdx = nextIdx
	}
	if len(points) > 0 {
		points = points[:len(points)-1]
	}
	return points
}

func (ctx *routeContext) classify(point geo.Point) pointContext {
	key := fmt.Sprintf("%.4f,%.4f", point.Lat, point.Lon)
	if cached, ok := ctx.pointCache[key]; ok {
		return cached
	}
	geoCtx := geo.LookupPoint(point)
	classified := pointContext{
		geoContext: geoCtx,
		isLand: geoCtx.AirspaceOwner != "" &&
			geoCtx.SeaZoneOwner == "" &&
			geoCtx.SeaZoneType == geo.SeaZoneTypeNone &&
			!geoCtx.IsInternationalAirspace,
	}
	ctx.pointCache[key] = classified
	return classified
}

func (ctx *routeContext) traversalCost(from, to geo.Point) float64 {
	cost := heuristic(from, to)
	classified := ctx.classify(to)
	switch ctx.req.Domain {
	case enginev1.UnitDomain_DOMAIN_SEA:
		if classified.geoContext.IsInternationalWaters {
			return cost
		}
		if sim.CountryDisplayCode(classified.geoContext.SeaZoneOwner) != ctx.req.OwnerCountry {
			return cost * 1.5
		}
	case enginev1.UnitDomain_DOMAIN_LAND:
		if owner := sim.CountryDisplayCode(classified.geoContext.AirspaceOwner); owner != "" && owner != ctx.req.OwnerCountry {
			return cost * 1.5
		}
	case enginev1.UnitDomain_DOMAIN_AIR:
		if owner := sim.CountryDisplayCode(classified.geoContext.AirspaceOwner); owner != "" && owner != ctx.req.OwnerCountry {
			return cost * 1.2
		}
	}
	return cost
}

func (ctx *routeContext) pointPassable(point geo.Point) bool {
	classified := ctx.classify(point)
	switch ctx.req.Domain {
	case enginev1.UnitDomain_DOMAIN_LAND:
		if !classified.isLand {
			return false
		}
		owner := sim.CountryDisplayCode(classified.geoContext.AirspaceOwner)
		return owner == "" || owner == ctx.req.OwnerCountry || sim.GetRelationshipRuleWithCoalitions(ctx.req.RelationshipRules, ctx.req.CountryCoalitions, ctx.req.OwnerCountry, owner).AirspaceTransitAllowed
	case enginev1.UnitDomain_DOMAIN_SEA:
		if classified.isLand {
			return false
		}
		if classified.geoContext.IsTransitPassage {
			return true
		}
		owner := sim.CountryDisplayCode(classified.geoContext.SeaZoneOwner)
		if owner == "" || owner == ctx.req.OwnerCountry {
			return true
		}
		return sim.GetRelationshipRuleWithCoalitions(ctx.req.RelationshipRules, ctx.req.CountryCoalitions, ctx.req.OwnerCountry, owner).MaritimeTransitAllowed
	case enginev1.UnitDomain_DOMAIN_SUBSURFACE:
		return !classified.isLand
	case enginev1.UnitDomain_DOMAIN_AIR:
		if classified.geoContext.IsInternationalAirspace {
			return true
		}
		owner := sim.CountryDisplayCode(classified.geoContext.AirspaceOwner)
		if owner == "" || owner == ctx.req.OwnerCountry {
			return true
		}
		rule := sim.GetRelationshipRuleWithCoalitions(ctx.req.RelationshipRules, ctx.req.CountryCoalitions, ctx.req.OwnerCountry, owner)
		switch ctx.req.Purpose {
		case PurposeDefensiveAir:
			return rule.AirspaceTransitAllowed && rule.DefensivePositioningAllowed
		case PurposeStrike:
			return rule.AirspaceTransitAllowed && rule.AirspaceStrikeAllowed
		default:
			return rule.AirspaceTransitAllowed
		}
	default:
		return false
	}
}

func (ctx *routeContext) segmentPassable(start, end geo.Point) bool {
	steps := int(math.Ceil(maxFloat(math.Abs(end.Lat-start.Lat), math.Abs(end.Lon-start.Lon))/0.02)) + 1
	if steps < 4 {
		steps = 4
	}
	if steps > 512 {
		steps = 512
	}
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		point := geo.Point{
			Lat: start.Lat + (end.Lat-start.Lat)*t,
			Lon: start.Lon + (end.Lon-start.Lon)*t,
		}
		if !ctx.pointPassable(point) {
			return false
		}
	}
	return true
}

func (ctx *routeContext) explainBlockedPoint(point geo.Point) (string, string) {
	classified := ctx.classify(point)
	switch ctx.req.Domain {
	case enginev1.UnitDomain_DOMAIN_LAND:
		if !classified.isLand {
			return fmt.Sprintf("%s land units cannot route onto water", ctx.req.OwnerCountry), ""
		}
		owner := sim.CountryDisplayCode(classified.geoContext.AirspaceOwner)
		if owner != "" && owner != ctx.req.OwnerCountry &&
			!sim.GetRelationshipRuleWithCoalitions(ctx.req.RelationshipRules, ctx.req.CountryCoalitions, ctx.req.OwnerCountry, owner).AirspaceTransitAllowed {
			return fmt.Sprintf("%s cannot cross %s land border", ctx.req.OwnerCountry, owner), owner
		}
	case enginev1.UnitDomain_DOMAIN_SEA:
		if classified.isLand {
			return fmt.Sprintf("%s naval units cannot route onto land", ctx.req.OwnerCountry), ""
		}
		owner := sim.CountryDisplayCode(classified.geoContext.SeaZoneOwner)
		if owner != "" && owner != ctx.req.OwnerCountry &&
			!classified.geoContext.IsTransitPassage &&
			!sim.GetRelationshipRuleWithCoalitions(ctx.req.RelationshipRules, ctx.req.CountryCoalitions, ctx.req.OwnerCountry, owner).MaritimeTransitAllowed {
			return fmt.Sprintf("%s cannot transit %s territorial waters", ctx.req.OwnerCountry, owner), owner
		}
	case enginev1.UnitDomain_DOMAIN_SUBSURFACE:
		if classified.isLand {
			return fmt.Sprintf("%s submarines cannot route onto land", ctx.req.OwnerCountry), ""
		}
	case enginev1.UnitDomain_DOMAIN_AIR:
		owner := sim.CountryDisplayCode(classified.geoContext.AirspaceOwner)
		if owner != "" && owner != ctx.req.OwnerCountry {
			rule := sim.GetRelationshipRuleWithCoalitions(ctx.req.RelationshipRules, ctx.req.CountryCoalitions, ctx.req.OwnerCountry, owner)
			switch ctx.req.Purpose {
			case PurposeDefensiveAir:
				if !rule.AirspaceTransitAllowed || !rule.DefensivePositioningAllowed {
					return fmt.Sprintf("%s cannot conduct defensive air operations in %s airspace", ctx.req.OwnerCountry, owner), owner
				}
			case PurposeStrike:
				if !rule.AirspaceTransitAllowed || !rule.AirspaceStrikeAllowed {
					return fmt.Sprintf("%s cannot conduct strike operations in %s airspace", ctx.req.OwnerCountry, owner), owner
				}
			default:
				if !rule.AirspaceTransitAllowed {
					return fmt.Sprintf("%s cannot transit %s airspace", ctx.req.OwnerCountry, owner), owner
				}
			}
		}
	}
	return "", ""
}

func heuristic(a, b geo.Point) float64 {
	return math.Hypot(a.Lat-b.Lat, a.Lon-b.Lon)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
