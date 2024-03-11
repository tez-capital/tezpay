package cmd

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

func parseDateFlags(cmd *cobra.Command) (time.Time, time.Time, error) {
	startDateFlag, _ := cmd.Flags().GetString(START_DATE_FLAG)
	endDateFlag, _ := cmd.Flags().GetString(END_DATE_FLAG)
	monthFlag, _ := cmd.Flags().GetString(MONTH_FLAG)

	if startDateFlag != "" && endDateFlag != "" && monthFlag != "" {
		return time.Time{}, time.Time{}, errors.New("only start date and end date or month can be specified")
	}
	if startDateFlag != "" && endDateFlag != "" {
		startDate, err := time.Parse("2006-01-02", startDateFlag)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("failed to parse start date - %s", err)
		}
		endDate, err := time.Parse("2006-01-02", endDateFlag)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("failed to parse end date - %s", err)
		}
		if startDate.After(endDate) {
			return time.Time{}, time.Time{}, errors.New("start date cannot be after end date")
		}

		return startDate, endDate.Add(-time.Nanosecond), nil
	}
	if monthFlag != "" {
		month, err := time.Parse("2006-01", monthFlag)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("failed to parse month - %s", err)
		}
		startDate := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
		endDate := startDate.AddDate(0, 1, 0).Add(-time.Nanosecond)
		return startDate, endDate, nil
	}
	return time.Time{}, time.Time{}, errors.New("invalid date range")
}
