//go:build wasip1

// Command reducer is the wasip1 packaging of the pure inventory reducer.
//
// It is a WASI *command* module: it reads a JSON op-log from stdin, replays it
// through the pure, deterministic reducer.Apply, and writes the converged State
// JSON to stdout. Errors go to stderr with a non-zero exit.
//
// Why a command module (stdin->stdout) for the spike, not a //go:wasmexport
// reactor: it is the lowest-risk way to PROVE the Go<->JS determinism boundary
// (Mission A's whole purpose is to price that boundary before committing). The
// JS host (../../host/apply.mjs) runs this via node:wasi, feeding the full
// linearized log and capturing the byte-identical output. Swapping to an
// incremental //go:wasmexport apply() reactor wired directly into Autobase's
// apply() is the next step once the boundary is proven — see docs/MESH_PROGRESS.md.
//
// Build: GOOS=wasip1 GOARCH=wasm go build -o mesh/dist/reducer.wasm ./mesh/cmd/reducer
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"ph_holdings_app/mesh/reducer"
)

type input struct {
	Config reducer.Config `json:"config"` // Mission D: authorityPub enables capability enforcement
	Ops    []reducer.Op   `json:"ops"`
}

func main() {
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "reducer: read stdin:", err)
		os.Exit(1)
	}

	var in input
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &in); err != nil {
			fmt.Fprintln(os.Stderr, "reducer: parse ops:", err)
			os.Exit(2)
		}
	}

	state := reducer.ApplyWithConfig(in.Config, in.Ops)

	out, err := json.Marshal(state)
	if err != nil {
		fmt.Fprintln(os.Stderr, "reducer: marshal state:", err)
		os.Exit(3)
	}
	if _, err := os.Stdout.Write(out); err != nil {
		fmt.Fprintln(os.Stderr, "reducer: write stdout:", err)
		os.Exit(4)
	}
}
