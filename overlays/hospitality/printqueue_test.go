package hospitality

// Wave 5 C.2 — the print spooler. Documents enqueue jobs atomically with
// their own commit; a worker claims FIFO per target and marks printed or
// failed (with optional requeue to the tail).

import (
	"testing"
)

func TestPrintQueue_DocumentsEnqueueJobs(t *testing.T) {
	h := newHarness(t)
	inv := h.runSession(t, "T1", map[string]float64{"Kunafa": 1})

	var jobs []PrintJob
	if err := h.db.Order("id ASC").Find(&jobs).Error; err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 2 {
		t.Fatalf("expected a KOT job and an invoice job, got %d", len(jobs))
	}
	if jobs[0].Kind != PrintKindKitchenTicket || jobs[0].Target != "kitchen" {
		t.Fatalf("first job should be the kitchen ticket: %+v", jobs[0])
	}
	if jobs[1].Kind != PrintKindInvoice || jobs[1].PayloadRef != inv.Number {
		t.Fatalf("second job should reference invoice %s: %+v", inv.Number, jobs[1])
	}
	for _, j := range jobs {
		if j.Status != PrintQueued {
			t.Fatalf("jobs must start queued: %+v", j)
		}
	}
}

func TestPrintQueue_SplitEnqueuesOneJobPerInvoice(t *testing.T) {
	h := newHarness(t)
	sessionID, lines := openSessionWithLines(t, h, "T1", map[string]float64{"Karak Chai": 1, "Kunafa": 1})
	if _, err := h.svc.SplitSession(sessionID, [][]uint{{lines["Karak Chai"].ID}, {lines["Kunafa"].ID}}, h.manager); err != nil {
		t.Fatal(err)
	}
	var invoiceJobs int64
	if err := h.db.Model(&PrintJob{}).Where("kind = ?", PrintKindInvoice).Count(&invoiceJobs).Error; err != nil {
		t.Fatal(err)
	}
	if invoiceJobs != 2 {
		t.Fatalf("a 2-way split must spool 2 invoice jobs, got %d", invoiceJobs)
	}
}

func TestPrintQueue_WorkerClaimAndMark(t *testing.T) {
	h := newHarness(t)
	h.runSession(t, "T1", map[string]float64{"Kunafa": 1})

	// Claim per target: the kitchen worker only sees kitchen jobs.
	kot, err := h.svc.ClaimNextPrintJob("kitchen")
	if err != nil || kot == nil || kot.Kind != PrintKindKitchenTicket {
		t.Fatalf("kitchen claim: %+v err %v", kot, err)
	}
	if err := h.svc.MarkPrintJobPrinted(kot.ID); err != nil {
		t.Fatalf("mark printed: %v", err)
	}
	if err := h.svc.MarkPrintJobPrinted(kot.ID); err == nil {
		t.Fatal("double-print must be refused")
	}

	// The counter job fails once and requeues to the tail…
	counter, err := h.svc.ClaimNextPrintJob("counter")
	if err != nil || counter == nil {
		t.Fatalf("counter claim: %+v err %v", counter, err)
	}
	if err := h.svc.MarkPrintJobFailed(counter.ID, "paper out", true); err != nil {
		t.Fatalf("fail+requeue: %v", err)
	}
	again, err := h.svc.ClaimNextPrintJob("counter")
	if err != nil || again == nil || again.ID != counter.ID {
		t.Fatalf("requeued job must be claimable again: %+v err %v", again, err)
	}
	// …then fails terminally.
	if err := h.svc.MarkPrintJobFailed(again.ID, "printer dead", false); err != nil {
		t.Fatalf("terminal fail: %v", err)
	}
	var final PrintJob
	if err := h.db.First(&final, again.ID).Error; err != nil {
		t.Fatal(err)
	}
	if final.Status != PrintFailed || final.Attempts != 2 || final.Error != "printer dead" {
		t.Fatalf("unexpected terminal state: %+v", final)
	}

	// Queue drained.
	none, err := h.svc.ClaimNextPrintJob("")
	if err != nil || none != nil {
		t.Fatalf("queue should be empty: %+v err %v", none, err)
	}
}
