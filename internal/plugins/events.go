package plugins

type LoadedEvent struct {
	Plugin Plugin
}

type UnloadedEvent struct {
	Plugin Plugin
}
