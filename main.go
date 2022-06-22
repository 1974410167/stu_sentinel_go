// 1. 把每一个服务的限流规则写入map然后json序列化为字符串存入nacos
// 2. 通过nacos拿到json字符串，反序列化为sentinel配置
// 3. 运行
package main

import (
	"GolandProjects/stu_sentinel_go/sen1"
	"encoding/json"
	"fmt"
	sentinel "github.com/alibaba/sentinel-golang/api"
	"github.com/alibaba/sentinel-golang/core/base"
	"github.com/alibaba/sentinel-golang/core/flow"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"sync"
)

var w sync.WaitGroup

// 发布两个配置
func addConfigToNacos() {
	client := sen1.NacosConfigClient()
	// server1， 每秒只允许三个流量
	senConfig1 := make(map[string]any)
	senConfig1["Resource"] = "server1"
	senConfig1["Threshold"] = 3
	senConfig1["StatIntervalInMs"] = 1000
	mJson1, err := json.Marshal(senConfig1)
	if err != nil {
		panic(err.Error())
	}
	res, err := client.PublishConfig(vo.ConfigParam{
		DataId:  "server1",
		Group:   "sen_group",
		Content: string(mJson1)},
	)
	if err != nil {
		panic(err.Error())
	}
	fmt.Println(res)

	// server2， 每秒只允许5个流量
	senConfig2 := make(map[string]any)
	senConfig2["Resource"] = "server2"
	senConfig2["Threshold"] = 5
	senConfig2["StatIntervalInMs"] = 1000

	mJson2, err := json.Marshal(senConfig2)
	if err != nil {
		panic(err.Error())
	}
	res1, err := client.PublishConfig(vo.ConfigParam{
		DataId:  "server2",
		Group:   "sen_group",
		Content: string(mJson2)},
	)
	if err != nil {
		panic(err.Error())
	}
	fmt.Println(res1)
}

// 得到配置
func getConfigFromNacos(dataId string, group string) map[string]any {
	client := sen1.NacosConfigClient()
	content, err := client.GetConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  group})
	if err != nil {
		panic(err.Error())
	}
	m := make(map[string]any)
	json.Unmarshal([]byte(content), &m)
	return m
}

var Rules = []*flow.Rule{}

func addConfigInSentienl(m map[string]any) {
	resource := m["Resource"].(string)
	threshold := m["Threshold"].(float64)
	statIntervalInMs := m["StatIntervalInMs"].(float64)

	Rules = append(Rules, &flow.Rule{
		Resource: resource,
		// Threshold + StatIntervalInMs 可组合出多长时间限制通过多少请求，这里相当于限制为 2 qps
		Threshold:        threshold,
		StatIntervalInMs: uint32(statIntervalInMs),
		// 暂时不用关注这些参数
		TokenCalculateStrategy: flow.Direct,
		ControlBehavior:        flow.Reject,
	})
}

func Sen(resource string) {
	if err := sentinel.InitDefault(); err != nil {
		// 初始化失败
		panic(err.Error())
	}

	// 资源名
	// 加载流控规则，这里可以从nacos里边拿，也就是一个给每个服务都配置一个规则
	_, err := flow.LoadRules(Rules)
	if err != nil {
		panic(err.Error())
	}
	currency := 10
	w.Add(currency)
	for i := 0; i < currency; i++ {
		go func() {
			e, b := sentinel.Entry(resource, sentinel.WithTrafficType(base.Inbound))
			if b != nil {
				// 被流控
				fmt.Printf("blocked %s \n", b.BlockMsg())
			} else {
				// 通过
				fmt.Println("pass...")
				// 通过后必须调用Exit
				e.Exit()
			}
			w.Done()
		}()
	}
	w.Wait()
}

func main() {
	addConfigToNacos()
	config1 := getConfigFromNacos("server1", "sen_group")
	config2 := getConfigFromNacos("server2", "sen_group")
	addConfigInSentienl(config1)
	addConfigInSentienl(config2)
	Sen("server1")
	Sen("server2")
}
