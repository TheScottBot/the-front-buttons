package platform

import (
	"bytes"
	"encoding/xml"
)

// LaunchAgentPlist is the macOS per user login agent that starts the daemon
// (TE10), with every key from the local launchd.plist(5) man page: RunAtLoad
// starts it at login; KeepAlive with SuccessfulExit false restarts it after a
// crash but leaves it stopped after a clean shutdown; ProcessType Interactive
// lifts the background throttling, because key presses must feel immediate.
// Output from before the daemon's own log opens goes to logPath.
func LaunchAgentPlist(binaryPath string, logPath string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple Computer//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>` + escapeXML(LaunchAgentLabel) + `</string>
	<key>ProgramArguments</key>
	<array>
		<string>` + escapeXML(binaryPath) + `</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<dict>
		<key>SuccessfulExit</key>
		<false/>
	</dict>
	<key>ProcessType</key>
	<string>Interactive</string>
	<key>StandardOutPath</key>
	<string>` + escapeXML(logPath) + `</string>
	<key>StandardErrorPath</key>
	<string>` + escapeXML(logPath) + `</string>
</dict>
</plist>
`
}

// escapeXML escapes text for an XML element, so a path holding an ampersand or
// an angle bracket cannot break, or inject into, the generated file.
func escapeXML(text string) string {
	var escaped bytes.Buffer

	_ = xml.EscapeText(&escaped, []byte(text))

	return escaped.String()
}

// ScheduledTaskXML is the Windows scheduled task that starts the daemon when the
// given user logs on (TE10), from Microsoft's Task Scheduler schema
// (evaluation, TE10). It runs in the user's own interactive session at least
// privilege, which Microsoft documents a non administrator may register for
// their own account without a password; it runs on battery, has no time limit,
// and never starts a second copy.
func ScheduledTaskXML(binaryPath string, userID string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<Task version="1.3" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo>
    <Description>` + escapeXML(ProductDisplayName) + `: drives the Stream Deck.</Description>
  </RegistrationInfo>
  <Triggers>
    <LogonTrigger>
      <Enabled>true</Enabled>
      <UserId>` + escapeXML(userID) + `</UserId>
    </LogonTrigger>
  </Triggers>
  <Principals>
    <Principal id="Author">
      <UserId>` + escapeXML(userID) + `</UserId>
      <LogonType>InteractiveToken</LogonType>
      <RunLevel>LeastPrivilege</RunLevel>
    </Principal>
  </Principals>
  <Settings>
    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>
    <StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>
    <ExecutionTimeLimit>PT0S</ExecutionTimeLimit>
    <Enabled>true</Enabled>
  </Settings>
  <Actions Context="Author">
    <Exec>
      <Command>` + escapeXML(binaryPath) + `</Command>
    </Exec>
  </Actions>
</Task>
`
}
