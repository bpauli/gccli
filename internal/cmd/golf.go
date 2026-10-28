package cmd

var golfPageParams = Params{"perPage": In("Limit"), "page": Page}

var golfGroup = Group{Name: "golf", Help: "Show golf scorecards, courses, stats, clubs and leaderboards.", Children: []Node{
	Leaf{
		Name: "scorecards", Help: "List your golf scorecards.", Default: true,
		Spec: argsOf(&struct {
			AlignedPagination `embed:"" set:"items=scorecards"`
		}{}),
		Endpoint: "golf.scorecards", Params: golfPageParams,
		Verb: "list golf scorecards", Output: JSON{},
	},
	Leaf{
		Name: "courses", Help: "List the golf courses you played.",
		Spec: argsOf(&struct {
			AlignedPagination `embed:"" set:"items=courses"`
		}{}),
		Endpoint: "golf.courses", Params: golfPageParams,
		Verb: "list golf courses", Output: JSON{},
	},
	Leaf{
		Name: "course", Help: "Show one golf course by its course snapshot ID.",
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"Course snapshot ID."`
		}{}),
		Endpoint: "golf.course", Params: Params{"id": In("ID")},
		Verb: "get golf course", Output: JSON{List: true},
	},
	Leaf{Name: "stats", Help: "Show your golf statistics.", Endpoint: "golf.stats", Verb: "get golf stats", Output: JSON{}},
	Leaf{
		Name: "shot-stats", Help: "Show your golf shot statistics of one shot type.",
		Spec: argsOf(&struct {
			Kind string `arg:"" help:"Shot type: drive, approach, chip or putt." enum:"drive,approach,chip,putt"`
		}{}),
		Endpoint: "golf.shot-stats", Params: Params{"kind": In("Kind")},
		Verb: "get golf shot stats", Output: JSON{},
	},
	Leaf{Name: "clubs", Help: "List the golf clubs in your bag.", Endpoint: "golf.clubs", Verb: "list golf clubs", Output: JSON{List: true}},
	Leaf{Name: "club-types", Help: "List the golf club types with their loft and lie angles.", Endpoint: "golf.club-types", Verb: "list golf club types", Output: JSON{List: true}},
	Leaf{Name: "club-flex-types", Help: "List the golf club shaft flex types.", Endpoint: "golf.club-flex-types", Verb: "list golf club flex types", Output: JSON{List: true}},
	Leaf{
		Name: "leaderboards", Help: "List the golf leaderboards you played in.",
		Spec: argsOf(&struct {
			AlignedPagination `embed:"" set:"items=leaderboards"`
		}{}),
		Endpoint: "golf.leaderboards", Params: golfPageParams,
		Verb: "list golf leaderboards", Output: JSON{},
	},
}}
