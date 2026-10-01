package lobby

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"spooknloot/pkg/sim"
)

func chatTexts(l *Lobby) []string {
	var out []string
	for _, c := range l.Chat() {
		if c.System {
			out = append(out, "* "+c.Text)
		} else {
			out = append(out, c.Name+": "+c.Text)
		}
	}
	return out
}

func hasLine(l *Lobby, line string) bool {
	for _, t := range chatTexts(l) {
		if t == line {
			return true
		}
	}
	return false
}

func TestChat(t *testing.T) {
	host, err := newHost("Host", sim.ClassWarrior, "127.0.0.1", testPort+5, false)
	if err != nil {
		t.Fatal(err)
	}
	defer host.Leave()
	addr := fmt.Sprintf("127.0.0.1:%d", testPort+5)
	a := Join(addr, "A", sim.ClassWarrior)
	defer a.Leave()
	all := []*Lobby{host, a}
	waitFor(t, "a joined", all, func() bool { return hasLine(a, "* A joined") && hasLine(host, "* A joined") })

	b := Join(addr, "B", sim.ClassWarrior)
	all = append(all, b)
	waitFor(t, "b joined", all, func() bool { return hasLine(a, "* B joined") })

	a.SendChat("  hello  ")
	host.SendChat("hi from the host")
	waitFor(t, "messages everywhere", all, func() bool {
		for _, l := range all {
			if !hasLine(l, "A: hello") || !hasLine(l, "Host: hi from the host") {
				return false
			}
		}
		return true
	})

	// Characters the font can't show are replaced; long lines are cut.
	b.SendChat("grüße " + strings.Repeat("x", 200))
	waitFor(t, "sanitized message", all, func() bool {
		for _, c := range a.Chat() {
			if c.Name == "B" {
				return strings.HasPrefix(c.Text, "gr??e x") && len(c.Text) == MaxChatLength
			}
		}
		return false
	})

	// Spam is limited per player.
	for i := 0; i < 10; i++ {
		a.SendChat(fmt.Sprintf("spam %d", i))
	}
	// "hello" already used one of A's five messages in this window, so
	// spam 0-3 get through. Wait for the last one, then give a possible
	// fifth one time to arrive.
	waitFor(t, "allowed spam delivered", all, func() bool { return hasLine(b, "A: spam 3") })
	deadline := time.Now().Add(300 * time.Millisecond)
	for time.Now().Before(deadline) {
		for _, l := range all {
			l.Update()
		}
		time.Sleep(10 * time.Millisecond)
	}
	spam := 0
	for _, line := range chatTexts(b) {
		if strings.HasPrefix(line, "A: spam") {
			spam++
		}
	}

	if spam != chatRateLimit-1 {
		t.Errorf("%d spam messages got through, want %d", spam, chatRateLimit-1)
	}

	b.Leave()
	waitFor(t, "b left", all, func() bool { return hasLine(a, "* B left") })

	if SanitizeChat("   ") != "" {
		t.Error("blank message not dropped")
	}
}
