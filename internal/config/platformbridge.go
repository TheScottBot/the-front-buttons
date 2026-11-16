package config

import "go.yaml.in/yaml/v4"

// decodePlatformBridge reads the platform bridge: cgo or purego, written
// exactly. Anything else, empty text included, is refused, since leaving the
// field out already means cgo.
func decodePlatformBridge(bridgeNode *yaml.Node, list *problemList) PlatformBridge {
	text, isText := textOf(bridgeNode)
	if !isText {
		list.addf(bridgeNode, platformBridgeName, ReasonWrongType, "expected text, %q or %q",
			PlatformBridgeCgo, PlatformBridgePurego)

		return ""
	}

	switch bridge := PlatformBridge(text); bridge {
	case PlatformBridgeCgo, PlatformBridgePurego:
		return bridge
	default:
		list.addf(bridgeNode, platformBridgeName, ReasonUnknownBridge, "must be %q or %q",
			PlatformBridgeCgo, PlatformBridgePurego)

		return ""
	}
}
