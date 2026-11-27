package report

// TargetType is the kind of resource a report is about, stored in
// reports.target_type.
type TargetType string

// Target types allowed by the report_target enum.
const (
	TargetPhoto       TargetType = "photo"
	TargetObservation TargetType = "observation"
	TargetTrail       TargetType = "trail"
	TargetUser        TargetType = "user"
)

// Valid reports whether t is a known target type.
func (t TargetType) Valid() bool {
	switch t {
	case TargetPhoto, TargetObservation, TargetTrail, TargetUser:
		return true
	}
	return false
}
