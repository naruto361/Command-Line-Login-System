package cli

import "time"

var istLocation = mustIST()

func mustIST() *time.Location {
	loc, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		return time.FixedZone("IST", 5*3600+30*60)
	}
	return loc
}

// formatIST renders timestamps in Indian Standard Time for CLI display.
// Values are still stored in UTC in the database.
func formatIST(t time.Time) string {
	return t.In(istLocation).Format("2006-01-02 15:04:05 IST")
}
