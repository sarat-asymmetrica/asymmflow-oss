package main

import "testing"

func TestBuildMachineIDHash_IsStableAcrossMACOrder(t *testing.T) {
	first := []string{
		"demo-workstation.local",
		"36:93:c7:af:e2:c4",
		"2e:de:24:60:b0:09",
		"darwin",
		"arm64",
	}
	second := []string{
		"demo-workstation.local",
		"2e:de:24:60:b0:09",
		"36:93:c7:af:e2:c4",
		"darwin",
		"arm64",
	}

	if got, want := buildMachineIDHash(first), buildMachineIDHash(second); got == want {
		t.Fatalf("expected different hashes before sorting, got identical %s", got)
	}

	sortedFirst := []string{
		"demo-workstation.local",
		"2e:de:24:60:b0:09",
		"36:93:c7:af:e2:c4",
		"darwin",
		"arm64",
	}
	sortedSecond := []string{
		"demo-workstation.local",
		"2e:de:24:60:b0:09",
		"36:93:c7:af:e2:c4",
		"darwin",
		"arm64",
	}

	if got, want := buildMachineIDHash(sortedFirst), buildMachineIDHash(sortedSecond); got != want {
		t.Fatalf("expected stable hash after sorting, got %s want %s", got, want)
	}
}

func TestGetMachineID_CurrentDeviceIsStable(t *testing.T) {
	first := GetMachineID()
	second := GetMachineID()
	if first == "" || second == "" {
		t.Fatal("expected non-empty machine id")
	}
	if first != second {
		t.Fatalf("expected stable machine id across consecutive calls, got %s and %s", first, second)
	}
	t.Logf("current_machine_id=%s", first)
}
