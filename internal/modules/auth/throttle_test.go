package auth

import (
	"testing"
	"time"
)

func TestLoginThrottle(t *testing.T) {
	now := time.Date(2026, 10, 12, 10, 0, 0, 0, time.UTC)
	th := NewLoginThrottle(3, 5, 15*time.Minute)
	th.now = func() time.Time { return now }

	for range 3 {
		if _, blocked := th.Blocked("asha", "10.0.0.1"); blocked {
			t.Fatal("blocked too early")
		}
		th.Failure("asha", "10.0.0.1")
	}
	wait, blocked := th.Blocked("asha", "10.0.0.1")
	if !blocked || wait != 15*time.Minute {
		t.Fatalf("after 3 failures: blocked=%v wait=%v", blocked, wait)
	}
	if _, blocked := th.Blocked("asha", "10.0.0.2"); blocked {
		t.Error("another IP must not be blocked for this account")
	}
	if _, blocked := th.Blocked("ravi", "10.0.0.1"); blocked {
		t.Error("another account from the same IP is below the IP limit")
	}

	// Two more failures on other accounts reach the per-IP limit.
	th.Failure("ravi", "10.0.0.1")
	th.Failure("meena", "10.0.0.1")
	if _, blocked := th.Blocked("new.user", "10.0.0.1"); !blocked {
		t.Error("per-IP limit must block every account from that IP")
	}

	now = now.Add(15 * time.Minute)
	if _, blocked := th.Blocked("asha", "10.0.0.1"); blocked {
		t.Error("window must expire")
	}
}

func TestLoginThrottleSuccessResetsAccountOnly(t *testing.T) {
	th := NewLoginThrottle(2, 3, time.Minute)
	th.Failure("asha", "ip")
	th.Failure("asha", "ip")
	th.Success("asha", "ip")
	if _, blocked := th.Blocked("asha", "ip"); blocked {
		t.Fatal("success must clear the account counter")
	}
	th.Failure("asha", "ip")
	if _, blocked := th.Blocked("asha", "ip"); !blocked {
		t.Fatal("per-IP failures must survive a successful sign-in")
	}
}
