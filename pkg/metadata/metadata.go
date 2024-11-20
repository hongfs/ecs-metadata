package metadata

import (
	"encoding/json"
	"errors"
	"go.uber.org/atomic"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

func init() {
	// 避免阿里云函数计算环境下运行后面内容
	if os.Getenv("FC_ACCOUNT_ID") != "" {
		return
	}

	go func() {
		for {
			if HasCacheRam.Load() {
				err := refreshRam()

				if err != nil {
					log.Printf("refresh ram error: %v\n", err)
				}

				// 如果没有数据，那就快点进入下一个循环读取
				if cacheRam.AccessKeyID == "" {
					time.Sleep(time.Second * 1)
					continue
				}
			}

			time.Sleep(time.Minute * 1)
		}
	}()
}

// Hostname 获取主机名
func Hostname() string {
	hostname, _ := request("hostname")
	return hostname
}

// ID 获取实例ID
func ID() string {
	id, _ := request("instance-id")
	return id
}

// Region 获取实例所在地域，如 cn-hangzhou
func Region() string {
	region, _ := request("region-id")
	return region
}

// Zone 获取实例所在可用区，如 cn-hangzhou-b
func Zone() string {
	zone, _ := request("zone-id")
	return zone
}

type RamInfo struct {
	AccessKeyID     string    `json:"AccessKeyId"`
	AccessKeySecret string    `json:"AccessKeySecret"`
	Expiration      time.Time `json:"Expiration"`
	SecurityToken   string    `json:"SecurityToken"`
	LastUpdated     time.Time `json:"LastUpdated"`
	Code            string    `json:"Code"`
	Error           error     `json:"Error"`
}

var ErrRamInfoNil = errors.New("ram info is nil")

func Ram(name string) *RamInfo {
	// 走缓存
	if HasCacheRam.Load() && cacheRam != nil && cacheRam.AccessKeyID != "" {
		return cacheRam
	}

	return loadRam(name)
}

// HasCacheRam 开启 RAM 数据缓存
var HasCacheRam = atomic.NewBool(false)

// HasEnvRam 开启 RAM 更新环境变量
var HasEnvRam = atomic.NewBool(false)

var cacheRam = &RamInfo{}

func refreshRam() error {
	if os.Getenv("FC_INSTANCE_ID") != "" {
		return nil
	}

	ram := loadRam("")

	if ram == nil {
		return errors.New("ram is nil")
	}

	if ram.Error != nil {
		return ram.Error
	}

	if ram.AccessKeyID == "" {
		return errors.New("ram access key id is empty")
	}

	cacheRam = ram

	if HasEnvRam.Load() {
		os.Setenv("ALIBABA_CLOUD_ACCESS_KEY_ID", ram.AccessKeyID)
		os.Setenv("ALIBABA_CLOUD_ACCESS_KEY_SECRET", ram.AccessKeySecret)
		os.Setenv("ALIBABA_CLOUD_SECURITY_TOKEN", ram.SecurityToken)
	}

	return nil
}

func loadRam(name string) *RamInfo {
	if name == "" {
		name, _ = request("ram/security-credentials/")
	}

	if name == "" {
		return &RamInfo{
			Error: errors.New("ram name is empty"),
		}
	}

	data, err := request("ram/security-credentials/" + name)

	if err != nil {
		return &RamInfo{
			Error: err,
		}
	}

	ram := RamInfo{}

	err = json.Unmarshal([]byte(data), &ram)

	if err != nil {
		return &RamInfo{
			Error: err,
		}
	}

	if ram.AccessKeyID == "" {
		return &RamInfo{
			Error: errors.New(ram.Code),
		}
	}

	return &ram
}

// TerminationTime 获取实例释放时间，仅抢占式适用
func TerminationTime() time.Time {
	defaultTime := time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)

	terminationTime, err := request("termination-time")

	if err != nil {
		return defaultTime
	}

	t, err := time.Parse("2006-01-02T15:04:05Z", terminationTime)

	if err != nil {
		return defaultTime
	}

	return t
}

func request(path string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, "http://100.100.100.200/latest/meta-data/"+path, nil)

	if err != nil {
		return "", err
	}

	client := http.Client{
		Timeout: 1 * time.Second,
	}

	resp, err := client.Do(req)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("request failed")
	}

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return "", err
	}

	return string(body), nil
}
