package reghost

import "fmt"

// Configuration errors
var (
	ErrMissingActiveRecord  = fmt.Errorf("activeRecord is missing")
	ErrNoRecords            = fmt.Errorf("no records defined")
	ErrActiveRecordNotFound = fmt.Errorf("activeRecord does not exist in records")
)

// ErrEmptyRecordSet indicates a record set has no records
type ErrEmptyRecordSet struct {
	Name string
}

func (e *ErrEmptyRecordSet) Error() string {
	return fmt.Sprintf("record set '%s' is empty", e.Name)
}

// ErrInvalidRecord indicates an invalid record in a record set
type ErrInvalidRecord struct {
	RecordSet string
	Index     int
	Reason    string
}

func (e *ErrInvalidRecord) Error() string {
	return fmt.Sprintf("invalid record in '%s' at index %d: %s", e.RecordSet, e.Index, e.Reason)
}
