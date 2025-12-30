// Package poa provides Power-of-Attorney functionality including sector taxonomy
package taxonomy

import "fmt"

// SectorCode represents ISIC Rev.4 / NACE Rev.2 industry sector codes
// as required by AAP002 Section B.2
type SectorCode string

const (
	// SectorAgriculture - Section A: Agriculture, forestry and fishing
	SectorAgriculture SectorCode = "A"
	// SectorMining - Section B: Mining and quarrying
	SectorMining SectorCode = "B"
	// SectorManufacturing - Section C: Manufacturing
	SectorManufacturing SectorCode = "C"
	// SectorElectricityGas - Section D: Electricity, gas, steam and air conditioning supply
	SectorElectricityGas SectorCode = "D"
	// SectorWaterSupply - Section E: Water supply; sewerage, waste management and remediation
	SectorWaterSupply SectorCode = "E"
	// SectorConstruction - Section F: Construction
	SectorConstruction SectorCode = "F"
	// SectorWholesaleRetail - Section G: Wholesale and retail trade; repair of motor vehicles
	SectorWholesaleRetail SectorCode = "G"
	// SectorTransportStorage - Section H: Transportation and storage
	SectorTransportStorage SectorCode = "H"
	// SectorAccommodationFood - Section I: Accommodation and food service activities
	SectorAccommodationFood SectorCode = "I"
	// SectorInfoCommunication - Section J: Information and communication
	SectorInfoCommunication SectorCode = "J"
	// SectorFinanceInsurance - Section K: Financial and insurance activities
	SectorFinanceInsurance SectorCode = "K"
	// SectorRealEstate - Section L: Real estate activities
	SectorRealEstate SectorCode = "L"
	// SectorProfessionalScience - Section M: Professional, scientific and technical activities
	SectorProfessionalScience SectorCode = "M"
	// SectorAdminSupport - Section N: Administrative and support service activities
	SectorAdminSupport SectorCode = "N"
	// SectorPublicAdmin - Section O: Public administration and defence; compulsory social security
	SectorPublicAdmin SectorCode = "O"
	// SectorEducation - Section P: Education
	SectorEducation SectorCode = "P"
	// SectorHealthSocialWork - Section Q: Human health and social work activities
	SectorHealthSocialWork SectorCode = "Q"
	// SectorArtsEntertainment - Section R: Arts, entertainment and recreation
	SectorArtsEntertainment SectorCode = "R"
	// SectorOtherServices - Section S: Other service activities
	SectorOtherServices SectorCode = "S"
	// SectorHouseholdActivities - Section T: Activities of households as employers
	SectorHouseholdActivities SectorCode = "T"
	// SectorExtraterritorial - Section U: Activities of extraterritorial organisations and bodies
	SectorExtraterritorial SectorCode = "U"
)

// IndustrySector represents a specific industry sector with ISIC/NACE classification
type IndustrySector struct {
	Code        SectorCode `json:"code"`        // ISIC/NACE section code
	Division    string     `json:"division"`    // 2-digit division code (optional)
	Group       string     `json:"group"`       // 3-digit group code (optional)
	Class       string     `json:"class"`       // 4-digit class code (optional)
	Description string     `json:"description"` // Human-readable description
	Authorized  bool       `json:"authorized"`  // Whether authorization applies to this sector
}

// SectorScope represents the collection of authorized sectors for a PoA
type SectorScope struct {
	Sectors       []IndustrySector `json:"sectors"`
	AllSectors    bool             `json:"all_sectors"`    // If true, all sectors authorized
	ExcludedCodes []SectorCode     `json:"excluded_codes"` // Explicitly excluded sectors
}

// SectorMetadata provides detailed information about each ISIC/NACE sector
var SectorMetadata = map[SectorCode]struct {
	Name        string
	Description string
	Examples    []string
}{
	SectorAgriculture: {
		Name:        "Agriculture, Forestry and Fishing",
		Description: "Crop and animal production, hunting and related service activities; forestry and logging; fishing and aquaculture",
		Examples:    []string{"farming", "forestry", "fishing", "hunting"},
	},
	SectorMining: {
		Name:        "Mining and Quarrying",
		Description: "Extraction of naturally occurring mineral solids, liquids, gases",
		Examples:    []string{"coal mining", "oil extraction", "metal ore mining", "stone quarrying"},
	},
	SectorManufacturing: {
		Name:        "Manufacturing",
		Description: "Physical or chemical transformation of materials, substances, or components into new products",
		Examples:    []string{"food processing", "textiles", "chemicals", "electronics", "automotive"},
	},
	SectorElectricityGas: {
		Name:        "Electricity, Gas, Steam and Air Conditioning Supply",
		Description: "Generation and distribution of electricity, gas, steam, and air conditioning",
		Examples:    []string{"power generation", "gas distribution", "steam supply"},
	},
	SectorWaterSupply: {
		Name:        "Water Supply; Sewerage, Waste Management",
		Description: "Water collection, treatment and supply; sewerage; waste collection, treatment and disposal",
		Examples:    []string{"water treatment", "waste management", "recycling", "sewerage"},
	},
	SectorConstruction: {
		Name:        "Construction",
		Description: "Building construction, civil engineering, and specialized construction activities",
		Examples:    []string{"building construction", "civil engineering", "electrical installation", "plumbing"},
	},
	SectorWholesaleRetail: {
		Name:        "Wholesale and Retail Trade; Repair of Motor Vehicles",
		Description: "Sale of goods without transformation, including motor vehicle repair",
		Examples:    []string{"wholesale trade", "retail trade", "vehicle repair", "motor vehicle sales"},
	},
	SectorTransportStorage: {
		Name:        "Transportation and Storage",
		Description: "Passenger and freight transport by rail, road, water, air; warehousing and support activities",
		Examples:    []string{"land transport", "water transport", "air transport", "warehousing", "postal services"},
	},
	SectorAccommodationFood: {
		Name:        "Accommodation and Food Service Activities",
		Description: "Short-term accommodation for visitors; food and beverage serving",
		Examples:    []string{"hotels", "restaurants", "catering", "bars"},
	},
	SectorInfoCommunication: {
		Name:        "Information and Communication",
		Description: "Publishing, broadcasting, telecommunications, IT services, and data processing",
		Examples:    []string{"software development", "telecommunications", "broadcasting", "data processing"},
	},
	SectorFinanceInsurance: {
		Name:        "Financial and Insurance Activities",
		Description: "Financial intermediation, insurance, reinsurance, pension funding, and related activities",
		Examples:    []string{"banking", "insurance", "investment funds", "securities trading"},
	},
	SectorRealEstate: {
		Name:        "Real Estate Activities",
		Description: "Acting as lessors, agents and/or brokers in real estate transactions",
		Examples:    []string{"property leasing", "real estate agencies", "property management"},
	},
	SectorProfessionalScience: {
		Name:        "Professional, Scientific and Technical Activities",
		Description: "Legal, accounting, consulting, architectural, engineering, R&D, and advertising services",
		Examples:    []string{"legal services", "accounting", "engineering", "research", "consulting", "advertising"},
	},
	SectorAdminSupport: {
		Name:        "Administrative and Support Service Activities",
		Description: "Routine support activities for day-to-day operations including office admin and security",
		Examples:    []string{"office administration", "security services", "cleaning", "call centers"},
	},
	SectorPublicAdmin: {
		Name:        "Public Administration and Defence; Social Security",
		Description: "General public administration activities, defence, and compulsory social security",
		Examples:    []string{"government services", "defence", "public order", "social security"},
	},
	SectorEducation: {
		Name:        "Education",
		Description: "Public and private education at all levels and for all subjects",
		Examples:    []string{"primary education", "secondary education", "higher education", "vocational training"},
	},
	SectorHealthSocialWork: {
		Name:        "Human Health and Social Work Activities",
		Description: "Human health activities, residential care, and social work without accommodation",
		Examples:    []string{"hospitals", "medical practice", "nursing care", "social work"},
	},
	SectorArtsEntertainment: {
		Name:        "Arts, Entertainment and Recreation",
		Description: "Creative, arts, entertainment, sports, amusement and recreation activities",
		Examples:    []string{"performing arts", "museums", "sports activities", "gambling", "amusement parks"},
	},
	SectorOtherServices: {
		Name:        "Other Service Activities",
		Description: "Repair of personal goods, personal services, and membership organizations",
		Examples:    []string{"repair services", "personal care", "religious organizations", "trade unions"},
	},
	SectorHouseholdActivities: {
		Name:        "Activities of Households as Employers",
		Description: "Households employing domestic personnel",
		Examples:    []string{"domestic staff employment"},
	},
	SectorExtraterritorial: {
		Name:        "Activities of Extraterritorial Organisations",
		Description: "International organizations and foreign embassies",
		Examples:    []string{"United Nations", "embassies", "international organizations"},
	},
}

// ValidateSectorCode checks if a sector code is valid according to ISIC Rev.4 / NACE Rev.2
func ValidateSectorCode(code SectorCode) error {
	if _, ok := SectorMetadata[code]; !ok {
		return fmt.Errorf("invalid sector code: %s (must be valid ISIC Rev.4 / NACE Rev.2 section A-U)", code)
	}
	return nil
}

// ValidateSectorScope ensures the sector scope is properly configured
func ValidateSectorScope(scope *SectorScope) error {
	if scope == nil {
		return fmt.Errorf("sector scope cannot be nil")
	}

	if scope.AllSectors {
		// If all sectors authorized, excluded codes must be valid
		for _, code := range scope.ExcludedCodes {
			if err := ValidateSectorCode(code); err != nil {
				return fmt.Errorf("invalid excluded sector code: %w", err)
			}
		}
		return nil
	}

	if len(scope.Sectors) == 0 {
		return fmt.Errorf("at least one sector must be specified when all_sectors is false")
	}

	// Validate each sector
	for i, sector := range scope.Sectors {
		if err := ValidateSectorCode(sector.Code); err != nil {
			return fmt.Errorf("sector %d: %w", i, err)
		}

		// Validate division code format if provided (2 digits)
		if sector.Division != "" && len(sector.Division) != 2 {
			return fmt.Errorf("sector %d: division code must be 2 digits, got %q", i, sector.Division)
		}

		// Validate group code format if provided (3 digits)
		if sector.Group != "" && len(sector.Group) != 3 {
			return fmt.Errorf("sector %d: group code must be 3 digits, got %q", i, sector.Group)
		}

		// Validate class code format if provided (4 digits)
		if sector.Class != "" && len(sector.Class) != 4 {
			return fmt.Errorf("sector %d: class code must be 4 digits, got %q", i, sector.Class)
		}
	}

	return nil
}

// IsSectorAuthorized checks if a given sector code is authorized in the scope
func IsSectorAuthorized(scope *SectorScope, code SectorCode) bool {
	if scope == nil {
		return false
	}

	// Check if sector is explicitly excluded
	for _, excluded := range scope.ExcludedCodes {
		if excluded == code {
			return false
		}
	}

	// If all sectors authorized (and not excluded), return true
	if scope.AllSectors {
		return true
	}

	// Check if sector is in authorized list
	for _, sector := range scope.Sectors {
		if sector.Code == code && sector.Authorized {
			return true
		}
	}

	return false
}

// GetSectorDescription returns the human-readable description for a sector code
func GetSectorDescription(code SectorCode) string {
	if meta, ok := SectorMetadata[code]; ok {
		return meta.Description
	}
	return "Unknown sector"
}

// GetAllSectorCodes returns all valid ISIC/NACE sector codes
func GetAllSectorCodes() []SectorCode {
	return []SectorCode{
		SectorAgriculture,
		SectorMining,
		SectorManufacturing,
		SectorElectricityGas,
		SectorWaterSupply,
		SectorConstruction,
		SectorWholesaleRetail,
		SectorTransportStorage,
		SectorAccommodationFood,
		SectorInfoCommunication,
		SectorFinanceInsurance,
		SectorRealEstate,
		SectorProfessionalScience,
		SectorAdminSupport,
		SectorPublicAdmin,
		SectorEducation,
		SectorHealthSocialWork,
		SectorArtsEntertainment,
		SectorOtherServices,
		SectorHouseholdActivities,
		SectorExtraterritorial,
	}
}
