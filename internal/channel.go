package internal

/*
Channel names the broadcast groups on the shared qpool bus.
Use these constants so misspelled routes fail at compile time.
*/
type Channel string

const (
	ChannelMessages Channel = "messages"
)

/*
String returns the channel name as a string.
*/
func (channel Channel) String() string {
	return string(channel)
}
