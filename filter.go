package main

import (
	"errors"
	"net/url"
	"time"
)

const dateLayout = "2006-01-02"

type DateRange struct {
	From string
	To   string
}

func parseRange(q url.Values, now time.Time) (DateRange, error) {
	period := q.Get("period")
	from := q.Get("from")
	to := q.Get("to")

	if period != "" && (from != "" || to != "") {
		return DateRange{}, errors.New("Use either period or from/to, not both")
	}

	today := now.UTC()

	switch period {
	case "week":
		return DateRange{From: today.AddDate(0, 0, -7).Format(dateLayout), To: today.Format(dateLayout)}, nil
	case "month":
		return DateRange{From: today.AddDate(0, -1, 0).Format(dateLayout), To: today.Format(dateLayout)}, nil
	case "3months":
		return DateRange{From: today.AddDate(0, -3, 0).Format(dateLayout), To: today.Format(dateLayout)}, nil
	case "":

	default:
		return DateRange{}, errors.New("period must be week, month or 3months")
	}

	if from == "" && to == "" {
		return DateRange{}, nil
	}
	if from == "" || to == "" {
		return DateRange{}, errors.New("both from and to are required")
	}

	f, err := time.Parse(dateLayout, from)
	if err != nil {
		return DateRange{}, errors.New("from must be in YYYY-MM-DD format")
	}
	t, err := time.Parse(dateLayout, to)
	if err != nil {
		return DateRange{}, errors.New("to must be in YYYY-MM-DD format")
	}
	if f.After(t) {
		return DateRange{}, errors.New("from must not be after to")
	}

	return DateRange{From: from, To: to}, nil
}
