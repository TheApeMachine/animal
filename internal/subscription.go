package internal

/*
Subscription binds one consumer to a broadcast channel under a unique name.
*/
type Subscription struct {
	Channel Channel
	Name    string
}

/*
Subscribe creates a new subscription to the given channel with the given name.
*/
func Subscribe(channel Channel, name string) Subscription {
	return Subscription{
		Channel: channel,
		Name:    name,
	}
}
