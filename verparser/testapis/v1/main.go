package testapis

type OldEntry struct{}

type Entry struct {
	Name     string
	Id       string
	OldField float64
}

type Client struct{}

type Methods interface {
	NewClient() (client Client)
	OldApi(name string, id string) (session string)
	TemporalSession() (session string)
}

type RemovedInterface interface {
}
