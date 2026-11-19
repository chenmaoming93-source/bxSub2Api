package observability

import (
	"fmt"
	"time"
)

type BucketPoint struct {
	Start time.Time
	End   time.Time
	Label string
}

func BuildBucketPlan(start, end time.Time, window Window, bucket Bucket, loc *time.Location) ([]BucketPoint, error) {
	if err := ValidateWindowBucket(window, bucket); err != nil {
		return nil, err
	}
	duration, bucketDuration, err := BucketDurations(window, bucket)
	if err != nil {
		return nil, err
	}
	start = start.UTC().Truncate(bucketDuration)
	end = end.UTC()
	if alignedEnd := end.Truncate(bucketDuration); end.After(alignedEnd) {
		end = alignedEnd.Add(bucketDuration)
	}
	if start.Add(duration).Before(end) {
		end = start.Add(duration)
	}
	out := make([]BucketPoint, 0, int(duration/bucketDuration))
	for cursor := start; cursor.Before(end); cursor = cursor.Add(bucketDuration) {
		out = append(out, BucketPoint{
			Start: cursor,
			End:   cursor.Add(bucketDuration),
			Label: formatBucketLabel(cursor.In(loc), bucket),
		})
	}
	return out, nil
}

func BuildRangeBucketPlan(start, end time.Time, bucket Bucket, loc *time.Location) ([]BucketPoint, error) {
	bucketDuration, err := BucketDuration(bucket)
	if err != nil {
		return nil, err
	}
	if !start.Before(end) {
		return nil, fmt.Errorf("start must be before end")
	}
	duration := end.Sub(start)
	count := int((duration + bucketDuration - 1) / bucketDuration)
	if count > MaxBucketPoints {
		return nil, fmt.Errorf("range produces %d buckets; maximum is %d", count, MaxBucketPoints)
	}
	end = end.UTC()
	if alignedEnd := end.Truncate(bucketDuration); end.After(alignedEnd) {
		end = alignedEnd.Add(bucketDuration)
	}
	start = end.Add(-time.Duration(count) * bucketDuration)
	out := make([]BucketPoint, 0, count)
	for cursor := start; cursor.Before(end); cursor = cursor.Add(bucketDuration) {
		out = append(out, BucketPoint{
			Start: cursor,
			End:   cursor.Add(bucketDuration),
			Label: formatBucketLabel(cursor.In(loc), bucket),
		})
	}
	return out, nil
}

func BucketDurations(window Window, bucket Bucket) (time.Duration, time.Duration, error) {
	windowDuration, err := WindowDuration(window)
	if err != nil {
		return 0, 0, err
	}
	bucketDuration, err := BucketDuration(bucket)
	if err != nil {
		return 0, 0, ValidateWindowBucket(window, bucket)
	}
	return windowDuration, bucketDuration, nil
}

func formatBucketLabel(value time.Time, bucket Bucket) string {
	switch bucket {
	case Bucket6h:
		return value.Format("01-02 15:04")
	default:
		return value.Format("15:04")
	}
}
