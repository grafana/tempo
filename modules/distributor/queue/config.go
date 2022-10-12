package queue

type Config struct {
	Name        string
	TenantID    string
	Size        int
	WorkerCount int
}
