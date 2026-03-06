package transport

// IngestEventRequest is the JSON payload received from an agent via POST /api/v1/events.
// This is separated from the domain model to allow independent evolution of the HTTP contract.
type IngestEventRequest struct {
	ID         string                 `json:"id"         binding:"required,uuid"`
	EventTime  string                 `json:"event_time"`
	Message    string                 `json:"message"    binding:"required"`
	Raw        string                 `json:"raw"        binding:"required"`
	InputType  string                 `json:"input_type"`
	Facility   string                 `json:"facility"`
	Severity   string                 `json:"severity"`
	Category   string                 `json:"category"`
	Hostname   string                 `json:"hostname"`
	SourceIP   string                 `json:"source_ip"`
	Normalized map[string]interface{} `json:"normalized"`
}
