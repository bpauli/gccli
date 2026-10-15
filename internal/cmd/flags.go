package cmd

type Pagination struct {
	Limit  int `help:"Maximum number of ${items} to return." default:"20" short:"l"`
	Offset int `help:"Number of ${items} to skip." default:"0"`
}
