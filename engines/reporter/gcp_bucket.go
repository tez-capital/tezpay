package reporter_engines

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"sort"

	"cloud.google.com/go/storage"
	"github.com/gocarina/gocsv"
	"github.com/samber/lo"
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants"
	"github.com/tez-capital/tezpay/utils"
)

type GCSReporter struct {
	client *storage.Client
	bucket string
	ctx    context.Context
}

// NewGCSReporter creates a GCS-backed reporter. The caller is responsible for
// providing a context and the target bucket name. Credentials are picked up by
// the default application credentials of the environment.
func NewGCSReporter(ctx context.Context, bucket string) (*GCSReporter, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}
	return &GCSReporter{
		client: client,
		bucket: bucket,
		ctx:    ctx,
	}, nil
}

// helper to read an object from GCS
func (engine *GCSReporter) readObject(objectPath string) ([]byte, error) {
	rc, err := engine.client.Bucket(engine.bucket).Object(objectPath).NewReader(engine.ctx)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

// helper to write an object to GCS
func (engine *GCSReporter) writeObject(objectPath string, data []byte, contentType string) error {
	w := engine.client.Bucket(engine.bucket).Object(objectPath).NewWriter(engine.ctx)
	if contentType != "" {
		w.ContentType = contentType
	}
	if _, err := w.Write(data); err != nil {
		w.Close()
		return err
	}
	return w.Close()
}

func (engine *GCSReporter) GetExistingReports(cycle int64) ([]common.PayoutReport, error) {
	sourceFile := path.Join(fmt.Sprintf("%d", cycle), constants.PAYOUT_REPORT_FILE_NAME)
	data, err := engine.readObject(sourceFile)
	if err != nil {
		return []common.PayoutReport{}, err
	}
	reports := make([]common.PayoutReport, 0)
	if err := gocsv.UnmarshalBytes(data, &reports); err != nil {
		return []common.PayoutReport{}, err
	}
	return reports, nil
}

func (engine *GCSReporter) ReportPayouts(payouts []common.PayoutReport) error {
	if len(payouts) == 0 {
		return nil
	}
	cyclesToBeWritten := lo.Uniq(lo.Map(payouts, func(pr common.PayoutReport, _ int) int64 { return pr.Cycle }))

	for _, cycle := range cyclesToBeWritten {
		targetFile := path.Join(fmt.Sprintf("%d", cycle), constants.PAYOUT_REPORT_FILE_NAME)
		// keep the exact sort logic from FsReporter (descending by Amount)
		sort.Slice(payouts, func(i, j int) bool { return !payouts[i].Amount.IsLess(payouts[j].Amount) })
		reports := lo.Filter(payouts, func(payout common.PayoutReport, _ int) bool { return payout.Cycle == cycle })
		csvData, err := gocsv.MarshalBytes(reports)
		if err != nil {
			return err
		}
		if err := engine.writeObject(targetFile, csvData, "text/csv"); err != nil {
			return err
		}
	}
	return nil
}

func (engine *GCSReporter) ReportInvalidPayouts(payouts []common.PayoutRecipe) error {
	invalid := utils.OnlyInvalidPayouts(payouts)
	if len(invalid) == 0 {
		return nil
	}
	cyclesToBeWritten := lo.Uniq(lo.Map(invalid, func(pr common.PayoutRecipe, _ int) int64 { return pr.Cycle }))
	for _, cycle := range cyclesToBeWritten {
		targetFile := path.Join(fmt.Sprintf("%d", cycle), constants.INVALID_REPORT_FILE_NAME)
		// convert to PayoutReport as FsReporter did
		reports := lo.Map(utils.FilterPayoutsByCycle(invalid, cycle), mapPayoutRecipeToPayoutReport)
		csvData, err := gocsv.MarshalBytes(reports)
		if err != nil {
			return err
		}
		if err := engine.writeObject(targetFile, csvData, "text/csv"); err != nil {
			return err
		}
	}
	return nil
}

func (engine *GCSReporter) ReportCycleSummary(summary common.CyclePayoutSummary) error {
	targetFile := path.Join(fmt.Sprintf("%d", summary.Cycle), constants.REPORT_SUMMARY_FILE_NAME)
	data, err := json.MarshalIndent(summary, "", "\t")
	if err != nil {
		return err
	}
	return engine.writeObject(targetFile, data, "application/json")
}

func (engine *GCSReporter) GetExistingCycleSummary(cycle int64) (*common.CyclePayoutSummary, error) {
	sourceFile := path.Join(fmt.Sprintf("%d", cycle), constants.REPORT_SUMMARY_FILE_NAME)
	data, err := engine.readObject(sourceFile)
	if err != nil {
		return nil, err
	}
	var summary common.CyclePayoutSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		return nil, err
	}
	return &summary, nil
}

func (engine *GCSReporter) Close() error {
	return engine.client.Close()
}
