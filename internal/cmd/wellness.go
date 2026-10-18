package cmd

var wellnessGroup = Group{Name: "wellness", Help: "View wellness data.", Children: []Node{
	Leaf{
		Name: "menstrual-cycle", Help: "Show menstrual cycle data.",
		Spec: argsOf(&struct {
			StartDate Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"start"`
			EndDate   Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"end"`
		}{}),
		Endpoint: "wellness.menstrual-cycle-data",
		Params:   Params{"startDate": In("StartDate"), "endDate": In("EndDate")},
		Verb:     "get menstrual cycle data", Output: JSON{},
	},
	Leaf{
		Name: "menstrual-summary", Help: "Show menstrual cycle summary.",
		Spec: argsOf(&struct {
			StartDate Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"start"`
			EndDate   Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"end"`
		}{}),
		Endpoint: "wellness.menstrual-cycle-summary",
		Params:   Params{"startDate": In("StartDate"), "endDate": In("EndDate")},
		Verb:     "get menstrual cycle summary", Output: JSON{},
	},
	Leaf{Name: "pregnancy-summary", Help: "Show pregnancy summary.", Endpoint: "wellness.pregnancy-summary", Verb: "get pregnancy summary", Output: JSON{}},
	wellnessJetLagGroup,
}}

var wellnessJetLagGroup = Group{Name: "jet-lag", Help: "View jet lag trips.", Children: []Node{
	Leaf{Name: "list", Help: "List jet lag trips.", Default: true, Endpoint: "wellness.jet-lag-trips", Verb: "get jet lag trips", Output: JSON{}},
}}
