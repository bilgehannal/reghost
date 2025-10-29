package reghost

// Config represents the complete configuration structure
type Config struct {
	ActiveRecord string              `yaml:"activeRecord"`
	Records      map[string][]Record `yaml:"records"`
}

// Record represents a single DNS record rule
type Record struct {
	Domain string `yaml:"domain"`
	IP     string `yaml:"ip"`
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.ActiveRecord == "" {
		return ErrMissingActiveRecord
	}

	if c.Records == nil || len(c.Records) == 0 {
		return ErrNoRecords
	}

	if _, exists := c.Records[c.ActiveRecord]; !exists {
		return ErrActiveRecordNotFound
	}

	// Validate each record
	for name, records := range c.Records {
		if len(records) == 0 {
			return &ErrEmptyRecordSet{Name: name}
		}

		for i, record := range records {
			if record.Domain == "" {
				return &ErrInvalidRecord{
					RecordSet: name,
					Index:     i,
					Reason:    "domain is empty",
				}
			}
			if record.IP == "" {
				return &ErrInvalidRecord{
					RecordSet: name,
					Index:     i,
					Reason:    "ip is empty",
				}
			}
		}
	}

	return nil
}

// GetActiveRecords returns the currently active record set
func (c *Config) GetActiveRecords() []Record {
	if records, ok := c.Records[c.ActiveRecord]; ok {
		return records
	}
	return nil
}
