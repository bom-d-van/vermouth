package testapis

type Entry struct {
	Name     string
	Id       int
	NewField string
}

type EntryItem struct {
	Id      int
	Content string
}

type Client struct{}

type Methods interface {
	NewClient() (client Client)
	OldApi(name string, id int) (session string)
	NewApi(id int) (session string)
}
