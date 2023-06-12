package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mayr0y/animated-octo-couscous.git/internal/pkg/metrics"
	"github.com/mayr0y/animated-octo-couscous.git/internal/pkg/storage"
	"github.com/sirupsen/logrus"
	"io"
	"log"
	"net/http"
	"time"
)

func SendMetrics(ctx context.Context, s storage.Store, serverAddress string) error {
	getContext, getCancel := context.WithTimeout(ctx, time.Duration(1)*time.Second)
	defer getCancel()

	url := fmt.Sprintf("http://%s/update/", serverAddress)

	metricMap, err := s.GetMetrics(getContext)
	if err != nil {
		logrus.Errorf("Error with get metrics: %v", err)
		return err
	}

	for _, v := range metricMap {
		err = createPostRequest(url, v)
		if err != nil {
			return fmt.Errorf("error create post request %v", err)
		}
	}
	return nil
}

func createPostRequest(url string, metric *metrics.Metrics) error {
	body, err := json.Marshal(metric)
	if err != nil {
		return fmt.Errorf("error encoding metric %v", err)
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err = gz.Write(body); err != nil {
		return fmt.Errorf("error %s", err)
	}

	gz.Close()

	req, err := http.NewRequest(http.MethodPost, url, &buf)
	if err != nil {
		return fmt.Errorf("error send request %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error client %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("status code not 200")
	}

	return nil
}

func SendMetricsBatch(ctx context.Context, s storage.Store, serverAddress string) error {
	getContext, getCancel := context.WithTimeout(ctx, time.Duration(1)*time.Second)
	defer getCancel()

	metricsMap, err := s.GetMetrics(getContext)
	if err != nil {
		logrus.Errorf("Some error ocured during metrics get: %q", err)
		return err
	}

	metricsBatch := make([]*metrics.Metrics, 0)
	for _, v := range metricsMap {
		metricsBatch = append(metricsBatch, v)
	}

	url := fmt.Sprintf("http://%s/updates/", serverAddress)

	if err = sendBatchJSON(url, metricsBatch); err != nil {
		return fmt.Errorf("error create post request %v", err)
	}
	return nil
}

func sendBatchJSON(url string, metricsBatch []*metrics.Metrics) error {
	var buf bytes.Buffer
	jsonEncoder := json.NewEncoder(&buf)

	if err := jsonEncoder.Encode(metricsBatch); err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, &buf)
	if err != nil {
		log.Println(err)

		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err)

		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server response: %s", resp.Status)
	}
	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		return err
	}
	err = resp.Body.Close()
	if err != nil {
		return err
	}

	return nil
}
