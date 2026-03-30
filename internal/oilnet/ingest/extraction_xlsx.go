package ingest

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/aressim/internal/oilnet"
)

type ExtractionWorkbookRow struct {
	UnitID           string
	UnitName         string
	FuelType         string
	CountryArea      string
	SubnationalUnit  string
	ProductionType   string
	Status           string
	Operator         string
	Latitude         float64
	Longitude        float64
	LocationAccuracy string
	OnshoreOffshore  string
	FieldOutlineWKT  string
	WikiProjectURL   string
	WikiFieldURL     string
	ProductionBPD    float64
	ReserveBbl       float64
	ParentProjectID  string
}

type ExtractionProjectRow struct {
	ProjectID          string
	ProjectName        string
	FuelType           string
	CountryArea        string
	SubnationalUnit    string
	ProductionType     string
	Status             string
	Operator           string
	Latitude           float64
	Longitude          float64
	LocationAccuracy   string
	OnshoreOffshore    string
	ProjectOutlineWKT  string
	UnitIDs            []string
	WikiProjectURL     string
	ProductionBPD      float64
	ReserveBbl         float64
}

type extractionWorkbookData struct {
	Fields   []ExtractionWorkbookRow
	Projects []ExtractionProjectRow
}

type xlsxWorkbook struct {
	Sheets []xlsxSheet `xml:"sheets>sheet"`
}

type xlsxSheet struct {
	Name string `xml:"name,attr"`
	RID  string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
}

type xlsxRelationships struct {
	Relationships []xlsxRelationship `xml:"Relationship"`
}

type xlsxRelationship struct {
	ID     string `xml:"Id,attr"`
	Target string `xml:"Target,attr"`
}

type xlsxSharedStrings struct {
	Items []xlsxSI `xml:"si"`
}

type xlsxSI struct {
	Text string     `xml:"t"`
	Runs []xlsxText `xml:"r>t"`
}

type xlsxText struct {
	Text string `xml:",chardata"`
}

type xlsxRow struct {
	Cells []xlsxCell `xml:"c"`
}

type xlsxCell struct {
	Ref  string         `xml:"r,attr"`
	Type string         `xml:"t,attr"`
	V    string         `xml:"v"`
	IS   *xlsxInlineStr `xml:"is"`
}

type xlsxInlineStr struct {
	Text string `xml:"t"`
}

// ParseExtractionWorkbook reads the GOGET workbook and extracts field-level rows.
func ParseExtractionWorkbook(path string) ([]ExtractionWorkbookRow, error) {
	data, err := loadExtractionWorkbookData(path)
	if err != nil {
		return nil, err
	}
	return data.Fields, nil
}

// LoadExtractionWorkbookGraph converts workbook rows into a project/field overlay graph.
func LoadExtractionWorkbookGraph(path string) (*oilnet.Graph, error) {
	data, err := loadExtractionWorkbookData(path)
	if err != nil {
		return nil, err
	}
	graph := &oilnet.Graph{
		ID:          "oilnet-extraction-overlay",
		Name:        "Global Extraction Overlay",
		Description: "Project and field extraction overlay parsed from the Global Oil and Gas Extraction Tracker workbook.",
		Version:     "0.2.0",
		View:        "global",
		Sources: []oilnet.SourceRef{{
			Name:         "Global Oil and Gas Extraction Tracker",
			Organization: "Global Energy Monitor",
			URL:          path,
			Confidence:   0.95,
			Notes:        "Parsed directly from local XLSX workbook project- and field-level sheets.",
		}},
	}

	for _, project := range data.Projects {
		if project.ProjectID == "" || project.ProjectName == "" {
			continue
		}
		graph.Nodes = append(graph.Nodes, oilnet.Node{
			ID:               "project-" + strings.ToLower(strings.TrimSpace(project.ProjectID)),
			Name:             project.ProjectName,
			Kind:             oilnet.NodeProject,
			CountryCode:      countryCodeFromName(project.CountryArea),
			Operator:         project.Operator,
			Lat:              project.Latitude,
			Lon:              project.Longitude,
			State:            workbookState(project.Status),
			PrimaryCommodity: oilnet.CommodityCrude,
			ProductionBPD:    project.ProductionBPD,
			ReserveBbl:       project.ReserveBbl,
			CurrentFlowBPD:   project.ProductionBPD,
			CapacityBPD:      project.ProductionBPD,
			ChildFieldIDs:    project.UnitIDs,
			OutlineRings:     parseWKTOutline(project.ProjectOutlineWKT),
			Tags:             appendStatusTag([]string{strings.ToLower(strings.TrimSpace(project.ProductionType)), strings.ToLower(strings.TrimSpace(project.OnshoreOffshore)), strings.ToLower(strings.TrimSpace(project.LocationAccuracy))}, project.Status),
			Sources: []oilnet.SourceRef{{
				Name:         project.ProjectName,
				Organization: "Global Energy Monitor",
				URL:          project.WikiProjectURL,
				Confidence:   0.95,
			}},
		})
	}

	for _, field := range data.Fields {
		if field.UnitID == "" || field.UnitName == "" {
			continue
		}
		graph.Nodes = append(graph.Nodes, oilnet.Node{
			ID:               "extract-" + strings.ToLower(strings.TrimSpace(field.UnitID)),
			Name:             field.UnitName,
			Kind:             oilnet.NodeExtractionSite,
			CountryCode:      countryCodeFromName(field.CountryArea),
			Operator:         field.Operator,
			Lat:              field.Latitude,
			Lon:              field.Longitude,
			State:            workbookState(field.Status),
			PrimaryCommodity: oilnet.CommodityCrude,
			ParentProjectID:  field.ParentProjectID,
			ProductionBPD:    field.ProductionBPD,
			ReserveBbl:       field.ReserveBbl,
			CurrentFlowBPD:   field.ProductionBPD,
			CapacityBPD:      field.ProductionBPD,
			OutlineRings:     parseWKTOutline(field.FieldOutlineWKT),
			Tags:             appendStatusTag([]string{strings.ToLower(strings.TrimSpace(field.ProductionType)), strings.ToLower(strings.TrimSpace(field.OnshoreOffshore)), strings.ToLower(strings.TrimSpace(field.LocationAccuracy))}, field.Status),
			Sources: []oilnet.SourceRef{{
				Name:         field.UnitName,
				Organization: "Global Energy Monitor",
				URL:          firstNonEmpty(field.WikiFieldURL, field.WikiProjectURL),
				Confidence:   0.95,
			}},
		})
	}

	if err := oilnet.ValidateGraph(graph); err != nil {
		return nil, err
	}
	return graph, nil
}

func loadExtractionWorkbookData(path string) (*extractionWorkbookData, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("open extraction workbook: %w", err)
	}
	defer zr.Close()

	fileMap := make(map[string]*zip.File, len(zr.File))
	for _, file := range zr.File {
		fileMap[file.Name] = file
	}
	sharedStrings, err := readSharedStrings(fileMap["xl/sharedStrings.xml"])
	if err != nil {
		return nil, err
	}
	sheetByName, err := workbookSheetFiles(fileMap)
	if err != nil {
		return nil, err
	}

	fieldMainRows, err := parseWorksheetRecords(fileMap[sheetByName["Field-level main data"]], sharedStrings)
	if err != nil {
		return nil, err
	}
	fieldProdRows, err := parseWorksheetRecords(fileMap[sheetByName["Field-level production data"]], sharedStrings)
	if err != nil {
		return nil, err
	}
	fieldReserveRows, err := parseWorksheetRecords(fileMap[sheetByName["Field-level reserves data"]], sharedStrings)
	if err != nil {
		return nil, err
	}
	projectMainRows, err := parseWorksheetRecords(fileMap[sheetByName["Project-level main data"]], sharedStrings)
	if err != nil {
		return nil, err
	}
	projectProdRows, err := parseWorksheetRecords(fileMap[sheetByName["Project-level production data"]], sharedStrings)
	if err != nil {
		return nil, err
	}
	projectReserveRows, err := parseWorksheetRecords(fileMap[sheetByName["Project-level reserves data "]], sharedStrings)
	if err != nil {
		// workbook keeps a trailing space in this tab name in current export
		projectReserveRows, err = parseWorksheetRecords(fileMap[sheetByName["Project-level reserves data"]], sharedStrings)
		if err != nil {
			return nil, err
		}
	}

	fieldProduction := productionByID(fieldProdRows, "Unit ID")
	fieldReserves := reservesByID(fieldReserveRows, "Unit ID")
	projectProduction := productionByID(projectProdRows, "Project ID")
	projectReserves := reservesByID(projectReserveRows, "Project ID")

	projects := make([]ExtractionProjectRow, 0, len(projectMainRows))
	projectWikiToID := map[string]string{}
	projectByID := map[string]*ExtractionProjectRow{}
	fieldToProjectID := map[string]string{}
	for _, row := range projectMainRows {
		projectID := strings.TrimSpace(row["Project ID"])
		projectName := strings.TrimSpace(row["Project Name"])
		if projectID == "" || projectName == "" {
			continue
		}
		fuel := strings.ToLower(strings.TrimSpace(row["Fuel type"]))
		if !isOilLikeFuel(fuel) {
			continue
		}
		project := ExtractionProjectRow{
			ProjectID:         projectID,
			ProjectName:       projectName,
			FuelType:          row["Fuel type"],
			CountryArea:       row["Country/Area"],
			SubnationalUnit:   row["Subnational unit"],
			ProductionType:    row["Production Type"],
			Status:            row["Status"],
			Operator:          row["Operator"],
			Latitude:          parseWorkbookFloat(row["Latitude"]),
			Longitude:         parseWorkbookFloat(row["Longitude"]),
			LocationAccuracy:  row["Location accuracy"],
			OnshoreOffshore:   row["Onshore/Offshore"],
			ProjectOutlineWKT: row["Project outline (WKT)"],
			UnitIDs:           splitUnitIDs(row["Units (list of IDs)"]),
			WikiProjectURL:    row["Wiki URL (project)"],
			ProductionBPD:     projectProduction[projectID],
			ReserveBbl:        projectReserves[projectID],
		}
		if project.Latitude == 0 && project.Longitude == 0 {
			project.Latitude, project.Longitude = centroidForWKT(project.ProjectOutlineWKT)
		}
		projects = append(projects, project)
		projectWikiToID[strings.TrimSpace(project.WikiProjectURL)] = projectID
		projectByID[projectID] = &projects[len(projects)-1]
		for _, unitID := range project.UnitIDs {
			fieldToProjectID[strings.TrimSpace(unitID)] = projectID
		}
	}

	fields := make([]ExtractionWorkbookRow, 0, len(fieldMainRows))
	for _, row := range fieldMainRows {
		unitID := strings.TrimSpace(row["Unit ID"])
		unitName := strings.TrimSpace(row["Unit Name"])
		if unitID == "" || unitName == "" {
			continue
		}
		lat := parseWorkbookFloat(row["Latitude"])
		lon := parseWorkbookFloat(row["Longitude"])
		if lat == 0 && lon == 0 {
			continue
		}
		fuel := strings.ToLower(strings.TrimSpace(row["Fuel type"]))
		if !isOilLikeFuel(fuel) {
			continue
		}
		parentProjectID := fieldToProjectID[unitID]
		if parentProjectID == "" {
			parentProjectID = projectWikiToID[strings.TrimSpace(row["Wiki URL (project)"])]
		}
		field := ExtractionWorkbookRow{
			UnitID:           unitID,
			UnitName:         unitName,
			FuelType:         row["Fuel type"],
			CountryArea:      row["Country/Area"],
			SubnationalUnit:  row["Subnational unit"],
			ProductionType:   row["Production Type"],
			Status:           row["Status"],
			Operator:         row["Operator"],
			Latitude:         lat,
			Longitude:        lon,
			LocationAccuracy: row["Location accuracy"],
			OnshoreOffshore:  row["Onshore/Offshore"],
			FieldOutlineWKT:  row["Field outline (WKT)"],
			WikiProjectURL:   row["Wiki URL (project)"],
			WikiFieldURL:     row["Wiki URL (field)"],
			ProductionBPD:    fieldProduction[unitID],
			ReserveBbl:       fieldReserves[unitID],
			ParentProjectID:  projectNodeID(parentProjectID),
		}
		fields = append(fields, field)
		if parentProjectID != "" {
			if project := projectByID[parentProjectID]; project != nil && !contains(project.UnitIDs, unitID) {
				project.UnitIDs = append(project.UnitIDs, unitID)
			}
		}
	}

	for i := range projects {
		for j := range projects[i].UnitIDs {
			projects[i].UnitIDs[j] = fieldNodeID(projects[i].UnitIDs[j])
		}
		if (projects[i].Latitude == 0 && projects[i].Longitude == 0) && len(projects[i].UnitIDs) > 0 {
			projects[i].Latitude, projects[i].Longitude = centroidForChildFields(projects[i].UnitIDs, fields)
		}
	}

	return &extractionWorkbookData{
		Fields:   fields,
		Projects: projects,
	}, nil
}

func workbookSheetFiles(fileMap map[string]*zip.File) (map[string]string, error) {
	workbookFile := fileMap["xl/workbook.xml"]
	if workbookFile == nil {
		return nil, fmt.Errorf("workbook.xml missing")
	}
	var wb xlsxWorkbook
	if err := readXML(workbookFile, &wb); err != nil {
		return nil, fmt.Errorf("read workbook.xml: %w", err)
	}
	var rels xlsxRelationships
	if err := readXML(fileMap["xl/_rels/workbook.xml.rels"], &rels); err != nil {
		return nil, fmt.Errorf("read workbook rels: %w", err)
	}
	relByID := make(map[string]string, len(rels.Relationships))
	for _, rel := range rels.Relationships {
		relByID[rel.ID] = rel.Target
	}
	out := map[string]string{}
	for _, sheet := range wb.Sheets {
		target := relByID[sheet.RID]
		if target == "" {
			continue
		}
		if !strings.HasPrefix(target, "xl/") {
			target = "xl/" + strings.TrimPrefix(target, "/")
		}
		out[sheet.Name] = target
	}
	return out, nil
}

func readSharedStrings(file *zip.File) ([]string, error) {
	if file == nil {
		return nil, nil
	}
	var shared xlsxSharedStrings
	if err := readXML(file, &shared); err != nil {
		return nil, fmt.Errorf("read shared strings: %w", err)
	}
	out := make([]string, 0, len(shared.Items))
	for _, item := range shared.Items {
		if item.Text != "" {
			out = append(out, item.Text)
			continue
		}
		var b strings.Builder
		for _, run := range item.Runs {
			b.WriteString(run.Text)
		}
		out = append(out, b.String())
	}
	return out, nil
}

func parseWorksheetRecords(file *zip.File, sharedStrings []string) ([]map[string]string, error) {
	if file == nil {
		return nil, fmt.Errorf("worksheet missing")
	}
	reader, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("open worksheet: %w", err)
	}
	defer reader.Close()

	decoder := xml.NewDecoder(reader)
	headers := map[int]string{}
	rows := make([]map[string]string, 0, 8192)
	rowIndex := 0
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decode worksheet token: %w", err)
		}
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "row" {
			continue
		}
		var row xlsxRow
		if err := decoder.DecodeElement(&row, &start); err != nil {
			return nil, fmt.Errorf("decode worksheet row: %w", err)
		}
		values := make(map[int]string, len(row.Cells))
		for _, cell := range row.Cells {
			values[excelColumnIndex(cell.Ref)] = cellString(cell, sharedStrings)
		}
		if rowIndex == 0 {
			for col, value := range values {
				headers[col] = strings.TrimSpace(value)
			}
			rowIndex++
			continue
		}
		record := make(map[string]string, len(headers))
		for col, header := range headers {
			record[header] = values[col]
		}
		rows = append(rows, record)
		rowIndex++
	}
	return rows, nil
}

func readXML(file *zip.File, out any) error {
	if file == nil {
		return fmt.Errorf("zip member missing")
	}
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return err
	}
	return xml.Unmarshal(data, out)
}

func excelColumnIndex(ref string) int {
	col := 0
	for _, ch := range ref {
		if ch < 'A' || ch > 'Z' {
			break
		}
		col = col*26 + int(ch-'A'+1)
	}
	return col
}

func cellString(cell xlsxCell, shared []string) string {
	if cell.Type == "inlineStr" && cell.IS != nil {
		return strings.TrimSpace(cell.IS.Text)
	}
	if cell.Type == "s" {
		index, err := strconv.Atoi(strings.TrimSpace(cell.V))
		if err != nil || index < 0 || index >= len(shared) {
			return ""
		}
		return strings.TrimSpace(shared[index])
	}
	return strings.TrimSpace(cell.V)
}

func productionByID(rows []map[string]string, idField string) map[string]float64 {
	out := map[string]float64{}
	bestYear := map[string]float64{}
	for _, row := range rows {
		id := strings.TrimSpace(row[idField])
		if id == "" || !isOilLikeFuel(strings.ToLower(strings.TrimSpace(row["Fuel description"]))) {
			continue
		}
		bpd := convertedProductionBPD(parseWorkbookFloat(row["Quantity (converted)"]), row["Units (converted)"])
		if bpd == 0 {
			continue
		}
		year := parseWorkbookFloat(row["Data Year"])
		if currentYear, ok := bestYear[id]; ok {
			if year < currentYear {
				continue
			}
			if year == currentYear {
				out[id] += bpd
				continue
			}
		}
		bestYear[id] = year
		out[id] = bpd
	}
	return out
}

func reservesByID(rows []map[string]string, idField string) map[string]float64 {
	out := map[string]float64{}
	bestYear := map[string]float64{}
	for _, row := range rows {
		id := strings.TrimSpace(row[idField])
		if id == "" || !isOilLikeFuel(strings.ToLower(strings.TrimSpace(row["Fuel description"]))) {
			continue
		}
		bbl := convertedReservesBbl(parseWorkbookFloat(row["Quantity (converted)"]), row["Units (converted)"])
		if bbl == 0 {
			continue
		}
		year := parseWorkbookFloat(firstNonEmpty(row["Data Year"], row["Status year"]))
		if currentYear, ok := bestYear[id]; ok {
			if year < currentYear {
				continue
			}
			if year == currentYear {
				out[id] += bbl
				continue
			}
		}
		bestYear[id] = year
		out[id] = bbl
	}
	return out
}

func convertedProductionBPD(quantity float64, units string) float64 {
	u := strings.ToLower(strings.TrimSpace(units))
	switch {
	case strings.Contains(u, "million bbl/y"):
		return quantity * 1_000_000 / 365.25
	case strings.Contains(u, "bbl/d"), strings.Contains(u, "bpd"):
		return quantity
	default:
		return 0
	}
}

func convertedReservesBbl(quantity float64, units string) float64 {
	u := strings.ToLower(strings.TrimSpace(units))
	switch {
	case strings.Contains(u, "million bbl"):
		return quantity * 1_000_000
	case strings.Contains(u, "bbl"):
		return quantity
	default:
		return 0
	}
}

func isOilLikeFuel(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(value, "oil") || strings.Contains(value, "condensate") || strings.Contains(value, "ngl")
}

func parseWorkbookFloat(raw string) float64 {
	cleaned := strings.TrimSpace(strings.ReplaceAll(raw, ",", ""))
	if cleaned == "" {
		return 0
	}
	value, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0
	}
	return value
}

func workbookState(status string) oilnet.OperationalState {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "operating", "operational", "producing":
		return oilnet.StateOperational
	case "in-development", "development", "construction", "pre-construction", "discovered", "exploration", "mothballed", "idle":
		return oilnet.StateDegraded
	case "closed", "retired", "cancelled", "canceled", "abandoned", "decommissioning", "underground gas storage":
		return oilnet.StateOffline
	default:
		return oilnet.StateOperational
	}
}

func appendStatusTag(tags []string, status string) []string {
	raw := strings.ToLower(strings.TrimSpace(status))
	if raw == "" {
		return tags
	}
	return append(tags, "status:"+raw)
}

func splitUnitIDs(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func parseWKTOutline(wkt string) [][]oilnet.RoutePoint {
	wkt = strings.TrimSpace(wkt)
	if wkt == "" {
		return nil
	}
	upper := strings.ToUpper(wkt)
	switch {
	case strings.HasPrefix(upper, "POLYGON"):
		body := strings.TrimSpace(wkt[len("POLYGON"):])
		body = strings.TrimPrefix(body, "((")
		body = strings.TrimSuffix(body, "))")
		return [][]oilnet.RoutePoint{parseWKTRing(strings.Split(body, "),(")[0])}
	case strings.HasPrefix(upper, "MULTIPOLYGON"):
		body := strings.TrimSpace(wkt[len("MULTIPOLYGON"):])
		body = strings.TrimPrefix(body, "(((")
		body = strings.TrimSuffix(body, ")))")
		polygons := strings.Split(body, ")),((")
		out := make([][]oilnet.RoutePoint, 0, len(polygons))
		for _, polygon := range polygons {
			ring := parseWKTRing(strings.Split(polygon, "),(")[0])
			if len(ring) > 2 {
				out = append(out, ring)
			}
		}
		return out
	default:
		return nil
	}
}

func parseWKTRing(raw string) []oilnet.RoutePoint {
	pairs := strings.Split(strings.TrimSpace(raw), ",")
	out := make([]oilnet.RoutePoint, 0, len(pairs))
	for _, pair := range pairs {
		fields := strings.Fields(strings.TrimSpace(pair))
		if len(fields) < 2 {
			continue
		}
		lon := parseWorkbookFloat(fields[0])
		lat := parseWorkbookFloat(fields[1])
		out = append(out, oilnet.RoutePoint{Lat: lat, Lon: lon})
	}
	return out
}

func centroidForWKT(wkt string) (float64, float64) {
	rings := parseWKTOutline(wkt)
	if len(rings) == 0 {
		return 0, 0
	}
	var latSum, lonSum float64
	var count float64
	for _, point := range rings[0] {
		latSum += point.Lat
		lonSum += point.Lon
		count++
	}
	if count == 0 {
		return 0, 0
	}
	return latSum / count, lonSum / count
}

func centroidForChildFields(fieldIDs []string, fields []ExtractionWorkbookRow) (float64, float64) {
	fieldByID := make(map[string]ExtractionWorkbookRow, len(fields))
	for _, field := range fields {
		fieldByID[fieldNodeID(field.UnitID)] = field
	}
	var latSum, lonSum float64
	var count float64
	for _, id := range fieldIDs {
		field, ok := fieldByID[id]
		if !ok {
			continue
		}
		latSum += field.Latitude
		lonSum += field.Longitude
		count++
	}
	if count == 0 {
		return 0, 0
	}
	return latSum / count, lonSum / count
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func projectNodeID(projectID string) string {
	projectID = strings.TrimSpace(strings.ToLower(projectID))
	if projectID == "" {
		return ""
	}
	return "project-" + projectID
}

func fieldNodeID(fieldID string) string {
	fieldID = strings.TrimSpace(strings.ToLower(fieldID))
	if fieldID == "" {
		return ""
	}
	return "extract-" + fieldID
}
