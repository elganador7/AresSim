package library

import "strings"

const DefaultValueOfStatisticalLifeUSD = 13_700_000

func DefaultReplacementCostUSD(assetClass string, domain, generalType int) float64 {
	assetClass = strings.TrimSpace(assetClass)
	switch assetClass {
	case "airbase":
		return 3_500_000_000
	case "port":
		return 1_800_000_000
	case "radar_site", "c2_site":
		return 350_000_000
	case "oil_field", "pipeline_node", "power_plant", "desalination_plant":
		return 900_000_000
	}
	switch domain {
	case 2:
		switch generalType {
		case 16, 17, 18:
			return 350_000_000
		case 13:
			return 600_000_000
		case 14, 15:
			return 180_000_000
		case 20, 21, 22:
			return 60_000_000
		case 30, 31, 32:
			return 10_000_000
		default:
			return 140_000_000
		}
	case 3:
		switch generalType {
		case 40:
			return 13_000_000_000
		case 41, 42:
			return 2_200_000_000
		case 43:
			return 900_000_000
		case 44:
			return 450_000_000
		case 45, 47:
			return 180_000_000
		case 46:
			return 4_000_000_000
		default:
			return 300_000_000
		}
	case 4:
		switch generalType {
		case 51, 52:
			return 7_000_000_000
		default:
			return 3_500_000_000
		}
	case 1:
		switch generalType {
		case 73:
			return 1_000_000_000
		case 72:
			return 250_000_000
		case 70, 71:
			return 45_000_000
		case 60:
			return 10_000_000
		case 61, 62, 63:
			return 5_000_000
		case 80, 81, 82, 83:
			return 20_000_000
		default:
			return 20_000_000
		}
	default:
		return 50_000_000
	}
}

func DefaultStrategicValueUSD(assetClass, targetClass string, domain, generalType int, employmentRole string) float64 {
	assetClass = strings.TrimSpace(assetClass)
	targetClass = strings.TrimSpace(targetClass)
	employmentRole = strings.TrimSpace(employmentRole)
	switch assetClass {
	case "airbase":
		return 6_000_000_000
	case "port":
		return 2_500_000_000
	case "radar_site":
		return 600_000_000
	case "c2_site":
		return 1_500_000_000
	case "power_plant":
		return 1_200_000_000
	case "desalination_plant":
		return 1_500_000_000
	case "oil_field", "pipeline_node":
		return 2_000_000_000
	}
	if targetClass == "runway" {
		return 3_000_000_000
	}
	switch domain {
	case 2:
		if generalType == 16 || generalType == 17 || generalType == 18 {
			return 800_000_000
		}
		return 300_000_000
	case 3, 4:
		return 500_000_000
	case 1:
		switch generalType {
		case 73:
			return 900_000_000
		case 72, 51, 52:
			return 1_100_000_000
		}
	}
	switch employmentRole {
	case "defensive":
		return 250_000_000
	case "offensive":
		return 400_000_000
	default:
		return 300_000_000
	}
}

func DefaultEconomicValueUSD(assetClass, affiliation string) float64 {
	assetClass = strings.TrimSpace(assetClass)
	affiliation = strings.TrimSpace(affiliation)
	switch assetClass {
	case "oil_field", "pipeline_node":
		return 5_000_000_000
	case "power_plant":
		return 3_000_000_000
	case "desalination_plant":
		return 4_000_000_000
	case "port":
		return 2_000_000_000
	case "airbase":
		return 500_000_000
	}
	switch affiliation {
	case "civilian":
		return 800_000_000
	case "dual_use":
		return 400_000_000
	default:
		return 0
	}
}

func DefaultAuthorizedPersonnel(assetClass string, domain, generalType int) int {
	assetClass = strings.TrimSpace(assetClass)
	switch assetClass {
	case "airbase":
		return 1500
	case "port":
		return 400
	case "radar_site":
		return 40
	case "c2_site":
		return 120
	case "oil_field", "pipeline_node":
		return 80
	case "power_plant":
		return 120
	case "desalination_plant":
		return 100
	}
	switch domain {
	case 2:
		switch generalType {
		case 10, 11, 12:
			return 1
		case 13:
			return 4
		case 14:
			return 6
		case 15:
			return 9
		case 16:
			return 15
		case 17:
			return 4
		case 18:
			return 8
		case 20:
			return 2
		case 21, 22:
			return 4
		default:
			return 1
		}
	case 3:
		switch generalType {
		case 40:
			return 5000
		case 41, 42:
			return 320
		case 43:
			return 180
		case 44:
			return 90
		case 45:
			return 40
		case 46:
			return 1000
		case 47:
			return 80
		default:
			return 60
		}
	case 4:
		switch generalType {
		case 51, 52:
			return 155
		default:
			return 135
		}
	case 1:
		switch generalType {
		case 60:
			return 4
		case 61:
			return 9
		case 62:
			return 10
		case 63:
			return 4
		case 70:
			return 5
		case 71:
			return 8
		case 72:
			return 6
		case 73:
			return 90
		case 80:
			return 12
		case 81, 82, 83:
			return 120
		case 90:
			return 40
		case 91:
			return 60
		case 92:
			return 30
		case 93:
			return 50
		case 94:
			return 30
		default:
			return 20
		}
	default:
		return 0
	}
}
