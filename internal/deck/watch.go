package deck

// ConnectionChanges is what changed between two enumerations.
type ConnectionChanges struct {
	Connected    []Connection
	Disconnected []Connection
}

// DiffConnections compares two enumerations by path. A deck unplugged and
// plugged back in, or moved to this machine by the dock, gets a new path, so it
// appears as one departure and one arrival.
func DiffConnections(previous []Connection, current []Connection) ConnectionChanges {
	changes := ConnectionChanges{Connected: nil, Disconnected: nil}

	previousPaths := make(map[string]bool, len(previous))
	for _, connection := range previous {
		previousPaths[connection.Path] = true
	}

	currentPaths := make(map[string]bool, len(current))
	for _, connection := range current {
		currentPaths[connection.Path] = true

		if !previousPaths[connection.Path] {
			changes.Connected = append(changes.Connected, connection)
		}
	}

	for _, connection := range previous {
		if !currentPaths[connection.Path] {
			changes.Disconnected = append(changes.Disconnected, connection)
		}
	}

	return changes
}

// SupportedConnections keeps only the devices in the model table.
func SupportedConnections(connections []Connection) []Connection {
	var supported []Connection

	for _, connection := range connections {
		if _, inTable := LookupModel(connection.VendorID, connection.ProductID); inTable {
			supported = append(supported, connection)
		}
	}

	return supported
}
