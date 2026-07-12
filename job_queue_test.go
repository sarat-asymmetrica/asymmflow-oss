package main

import (
	"context"
	"encoding/json"
	"testing"

	sqlite "github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
)

// TestJobQueueBasicFlow tests the job queue enqueue and retrieval
func TestJobQueueBasicFlow(t *testing.T) {
	// Setup in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Migrate Job table
	if err := db.AutoMigrate(&Job{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create job queue
	jq := NewJobQueue(db, 1)

	// Register test handler
	jq.RegisterHandler("test_job", func(ctx context.Context, job *Job) error {
		return nil
	})

	// Enqueue job
	input := map[string]string{"message": "Hello, World!"}
	jobID, err := jq.Enqueue("test_job", input, "test_user")
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	// Verify job was created
	job, err := jq.GetJob(jobID)
	if err != nil {
		t.Fatalf("Failed to get job status: %v", err)
	}

	// Verify initial state
	if job.Status != "pending" {
		t.Errorf("Expected status='pending', got %s", job.Status)
	}

	if job.Type != "test_job" {
		t.Errorf("Expected type='test_job', got %s", job.Type)
	}

	// Verify input was serialized
	var parsedInput map[string]string
	if err := json.Unmarshal([]byte(job.Input), &parsedInput); err != nil {
		t.Fatalf("Failed to parse job input: %v", err)
	}

	if parsedInput["message"] != "Hello, World!" {
		t.Errorf("Expected message='Hello, World!', got %s", parsedInput["message"])
	}
}

// TestJobQueueHandlerRegistration tests handler registration
func TestJobQueueHandlerRegistration(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&Job{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	jq := NewJobQueue(db, 1)

	// Register multiple handlers
	jq.RegisterHandler("job_type_a", func(ctx context.Context, job *Job) error {
		return nil
	})
	jq.RegisterHandler("job_type_b", func(ctx context.Context, job *Job) error {
		return nil
	})

	// Enqueue jobs of different types
	idA, err := jq.Enqueue("job_type_a", map[string]string{"key": "value_a"}, "test")
	if err != nil {
		t.Fatalf("Failed to enqueue job_type_a: %v", err)
	}

	idB, err := jq.Enqueue("job_type_b", map[string]string{"key": "value_b"}, "test")
	if err != nil {
		t.Fatalf("Failed to enqueue job_type_b: %v", err)
	}

	// Verify both jobs exist
	jobA, _ := jq.GetJob(idA)
	jobB, _ := jq.GetJob(idB)

	if jobA.Type != "job_type_a" {
		t.Errorf("Expected type='job_type_a', got %s", jobA.Type)
	}

	if jobB.Type != "job_type_b" {
		t.Errorf("Expected type='job_type_b', got %s", jobB.Type)
	}
}

// TestReportDataSerialization tests report data can be marshaled/unmarshaled
func TestReportDataSerialization(t *testing.T) {
	data := ReportData{
		WinRate:        0.35,
		ConversionRate: 0.21,
		AvgDealSize:    3000,
		Pipeline: []PipelineStage{
			{Stage: "RFQs", Count: 24, Value: 45000},
			{Stage: "Quoted", Count: 18, Value: 38000},
		},
	}

	// Marshal
	jsonData, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Failed to marshal report data: %v", err)
	}

	// Unmarshal
	var decoded ReportData
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal report data: %v", err)
	}

	// Verify
	if decoded.WinRate != 0.35 {
		t.Errorf("Expected WinRate=0.35, got %f", decoded.WinRate)
	}

	if len(decoded.Pipeline) != 2 {
		t.Errorf("Expected 2 pipeline stages, got %d", len(decoded.Pipeline))
	}
}

// BenchmarkJobQueue benchmarks job enqueue throughput
func BenchmarkJobQueue(b *testing.B) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&Job{})

	jq := NewJobQueue(db, 4)

	jq.RegisterHandler("bench_job", func(ctx context.Context, job *Job) error {
		return nil
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		jq.Enqueue("bench_job", map[string]string{}, "bench")
	}
}
