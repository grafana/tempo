package cos

import (
	"context"
	"encoding/xml"
	"fmt"
	"github.com/clbanning/mxj"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

type BucketPutOriginOptions struct {
	XMLName xml.Name           `xml:"OriginConfiguration"`
	Rule    []BucketOriginRule `xml:"OriginRule"`
}

type BucketOriginRule struct {
	RulePriority    int                    `xml:"RulePriority,omitempty"`
	OriginType      string                 `xml:"OriginType,omitempty"`
	OriginCondition *BucketOriginCondition `xml:"OriginCondition,omitempty"`
	OriginParameter *BucketOriginParameter `xml:"OriginParameter,omitempty"`
	OriginInfo      *BucketOriginInfo      `xml:"OriginInfo,omitempty"`
	HttpStandbyCode *HTTPStandbyCode       `xml:"HTTPStandbyCode,omitempty"`
}

type BucketOriginCondition struct {
	HTTPStatusCode string `xml:"HTTPStatusCode,omitempty"`
	Prefix         string `xml:"Prefix,omitempty"`
	Suffix         string `xml:"Suffix,omitempty"`
}

type BucketOriginParameter struct {
	Protocol                       string                          `xml:"Protocol,omitempty"`
	TransparentErrorCode           *bool                           `xml:"TransparentErrorCode,omitempty"`
	FollowQueryString              *bool                           `xml:"FollowQueryString,omitempty"`
	HttpHeader                     *BucketOriginHttpHeader         `xml:"HttpHeader,omitempty"`
	FollowRedirection              *bool                           `xml:"FollowRedirection,omitempty"`
	FollowRedirectionConfiguration *FollowRedirectionConfiguration `xml:"FollowRedirectionConfiguration,omitempty"`
	HttpRedirectCode               string                          `xml:"HttpRedirectCode,omitempty"`
	CopyOriginData                 *bool                           `xml:"CopyOriginData,omitempty"`
}

type FollowRedirectionConfiguration struct {
	FollowOriginHeaders *bool `xml:"FollowOriginHeaders,omitempty"`
	FollowUrlAutoDecode *bool `xml:"FollowUrlAutoDecode,omitempty"`
}

type BucketOriginHttpHeader struct {
	FollowAllHeaders    *bool              `xml:"FollowAllHeaders,omitempty"`
	NewHttpHeaders      []OriginHttpHeader `xml:"NewHttpHeaders>Header,omitempty"`
	FollowHttpHeaders   []OriginHttpHeader `xml:"FollowHttpHeaders>Header,omitempty"`
	ForbidFollowHeaders []OriginHttpHeader `xml:"ForbidFollowHttpHeaders>Header,omitempty"`
}

type OriginHttpHeader struct {
	Key   string `xml:"Key,omitempty"`
	Value string `xml:"Value,omitempty"`
}

type BucketOriginInfo struct {
	HostInfos []*BucketOriginHostInfo `xml:"HostInfo,omitempty"`
	FileInfo  *BucketOriginFileInfo   `xml:"FileInfo,omitempty"`
	// Deprecated: Use HostInfos instead.
	HostInfo *BucketOriginHostInfo `xml:"-"`
}

// StandbyHostName_N 已废弃, 请使用 StandbyHostName
// 使用 StandbyHostName 和 PrivateStandbyHost_N 时，需要指定StandbyHostName.Index和PrivateStandbyHost_N.Index, 表示备份源站的编号
type BucketOriginHostInfo struct {
	HostName string
	Weight   int64
	// Deprecated: Use StandbyHostName instead.
	StandbyHostName_N []string

	StandbyHostName      []*BucketOriginStandbyHost
	PrivateHost          *BucketOriginPrivateHost
	PrivateStandbyHost_N []*BucketOriginPrivateHost
}

type BucketOriginStandbyHost struct {
	Index    int64 `xml:"-"`
	HostName string
}

type BucketOriginPrivateHost struct {
	Index              int64                           `xml:"-"`
	Host               string                          `xml:"Host,omitempty"`
	CredentialProvider *BucketOriginCredentialProvider `xml:"CredentialProvider,omitempty"`
}
type BucketOriginCredentialProvider struct {
	AuthorizationAlgorithm string `xml:"AuthorizationAlgorithm,omitempty"`
	Region                 string `xml:"Region,omitempty"`
	SecretId               string `xml:"SecretId,omitempty"`
	SecretKey              string `xml:"SecretKey,omitempty"`
	EncryptedSecretKey     string `xml:"EncryptedSecretKey,omitempty"`
	Role                   string `xml:"Role,omitempty"`
}

type BucketOriginFileInfo struct {
	// 兼容旧版本
	PrefixDirective bool   `xml:"PrefixDirective,omitempty"`
	Prefix          string `xml:"Prefix,omitempty"`
	Suffix          string `xml:"Suffix,omitempty"`
	// 新版本
	PrefixConfiguration    *OriginPrefixConfiguration    `xml:"PrefixConfiguration,omitempty"`
	SuffixConfiguration    *OriginSuffixConfiguration    `xml:"SuffixConfiguration,omitempty"`
	FixedFileConfiguration *OriginFixedFileConfiguration `xml:"FixedFileConfiguration,omitempty"`
}

type OriginPrefixConfiguration struct {
	Prefix             string `xml:"Prefix,omitempty"`
	ReplacedWithPrefix string `xml:"ReplacedWithPrefix,omitempty"`
}

type OriginSuffixConfiguration struct {
	Suffix             string `xml:"Suffix,omitempty"`
	ReplacedWithSuffix string `xml:"ReplacedWithSuffix,omitempty"`
}

type OriginFixedFileConfiguration struct {
	FixedFilePath string `xml:"FixedFilePath,omitempty"`
}

type HTTPStandbyCode struct {
	StatusCode []string `xml:"StatusCode,omitempty"`
}

type BucketGetOriginResult BucketPutOriginOptions

func (this *BucketOriginInfo) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if this == nil {
		return nil
	}
	err := e.EncodeToken(start)
	if err != nil {
		return err
	}
	if len(this.HostInfos) > 0 {
		for _, hostInfo := range this.HostInfos {
			err = e.EncodeElement(hostInfo, xml.StartElement{Name: xml.Name{Local: "HostInfo"}})
			if err != nil {
				return err
			}
		}
	} else {
		if this.HostInfo != nil {
			err = e.EncodeElement(this.HostInfo, xml.StartElement{Name: xml.Name{Local: "HostInfo"}})
			if err != nil {
				return err
			}
		}
	}
	if this.FileInfo != nil {
		err = e.EncodeElement(this.FileInfo, xml.StartElement{Name: xml.Name{Local: "FileInfo"}})
		if err != nil {
			return err
		}
	}
	return e.EncodeToken(xml.EndElement{Name: start.Name})
}

func (this *BucketOriginInfo) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var val struct {
		XMLName   xml.Name                `xml:"OriginInfo"`
		HostInfos []*BucketOriginHostInfo `xml:"HostInfo,omitempty"`
		FileInfo  *BucketOriginFileInfo   `xml:"FileInfo,omitempty"`
	}
	err := d.DecodeElement(&val, &start)
	if err != nil {
		return err
	}
	this.HostInfos = val.HostInfos
	this.FileInfo = val.FileInfo
	if len(this.HostInfos) > 0 {
		this.HostInfo = this.HostInfos[0]
	}
	return nil
}

func (this *BucketOriginHostInfo) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if this == nil {
		return nil
	}
	err := e.EncodeToken(start)
	if err != nil {
		return err
	}
	if this.HostName != "" {
		err = e.EncodeElement(this.HostName, xml.StartElement{Name: xml.Name{Local: "HostName"}})
		if err != nil {
			return err
		}
	}
	if this.Weight != 0 {
		err = e.EncodeElement(this.Weight, xml.StartElement{Name: xml.Name{Local: "Weight"}})
		if err != nil {
			return err
		}
	}
	if this.PrivateHost != nil {
		err = e.EncodeElement(this.PrivateHost, xml.StartElement{Name: xml.Name{Local: "PrivateHost"}})
		if err != nil {
			return err
		}
	}
	// 优先使用 StandbyHostName
	if len(this.StandbyHostName) > 0 {
		for _, standby := range this.StandbyHostName {
			if standby.Index == 0 {
				return fmt.Errorf("The parameter Index must be set in StandbyHostName")
			}
			err = e.EncodeElement(standby.HostName, xml.StartElement{Name: xml.Name{Local: fmt.Sprintf("StandbyHostName_%v", standby.Index)}})
			if err != nil {
				return err
			}
		}
	} else {
		if len(this.StandbyHostName_N) > 0 && len(this.PrivateStandbyHost_N) > 0 {
			return fmt.Errorf("StandbyHostName_N and PrivateStandbyHost_N can not be both set, use StandbyHostName and PrivateStandbyHost_N instand")
		}
		for index, standByHostName := range this.StandbyHostName_N {
			err = e.EncodeElement(standByHostName, xml.StartElement{Name: xml.Name{Local: fmt.Sprintf("StandbyHostName_%v", index+1)}})
			if err != nil {
				return err
			}
		}
	}
	for _, standby := range this.PrivateStandbyHost_N {
		if standby.Index == 0 {
			return fmt.Errorf("The parameter Index must be set in PrivateStandbyHost_N")
		}
		err = e.EncodeElement(standby, xml.StartElement{Name: xml.Name{Local: fmt.Sprintf("PrivateStandbyHost_%v", standby.Index)}})
		if err != nil {
			return err
		}
	}
	return e.EncodeToken(xml.EndElement{Name: start.Name})
}

func (this *BucketOriginHostInfo) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var val struct {
		XMLName xml.Name
		Inner   []byte `xml:",innerxml"`
	}
	err := d.DecodeElement(&val, &start)
	if err != nil {
		return err
	}
	str := "<HostInfo>" + string(val.Inner) + "</HostInfo>"
	myMxjMap, err := mxj.NewMapXml([]byte(str))
	if err != nil {
		return err
	}
	myMap, ok := myMxjMap["HostInfo"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("XML HostInfo Parse failed")
	}

	var standbyHostList SortStandbyList
	var privateStandbyHostList SortStandbyList
	for key, value := range myMap {
		if key == "HostName" {
			if _, ok := value.(string); ok {
				this.HostName = value.(string)
			}
		}
		if key == "Weight" {
			if _, ok := value.(string); ok {
				v := value.(string)
				this.Weight, err = strconv.ParseInt(v, 10, 64)
				if err != nil {
					return err
				}
			}
		}
		if strings.HasPrefix(key, "StandbyHostName_") {
			if val, ok := value.(string); ok {
				indexStr := key[len("StandbyHostName_"):]
				index, err := strconv.ParseInt(indexStr, 10, 64)
				if err != nil {
					return fmt.Errorf("StandbyHostName Parse failed, node: %v", key)
				}
				standbyHostList = append(standbyHostList, SortStandby{
					Index:   index,
					Standby: val,
				})
			}
		}
		if key == "PrivateHost" {
			if _, ok := value.(map[string]interface{}); ok {
				this.PrivateHost = &BucketOriginPrivateHost{}
				err = mxj.Map(value.(map[string]interface{})).Struct(this.PrivateHost)
				if err != nil {
					return err
				}
			}
		}
		if strings.HasPrefix(key, "PrivateStandbyHost_") {
			if _, ok := value.(map[string]interface{}); ok {
				var privateStandbyHost_N BucketOriginPrivateHost
				err = mxj.Map(value.(map[string]interface{})).Struct(&privateStandbyHost_N)
				if err != nil {
					return err
				}
				indexStr := key[len("PrivateStandbyHost_"):]
				index, err := strconv.ParseInt(indexStr, 10, 64)
				if err != nil {
					return fmt.Errorf("PrivateStandbyHost Parse failed, node: %v", key)
				}
				privateStandbyHost_N.Index = index
				privateStandbyHostList = append(privateStandbyHostList, SortStandby{
					Index:          index,
					PrivateStandby: &privateStandbyHost_N,
				})
			}
		}
	}
	// 按顺序执行
	sort.Sort(standbyHostList)
	for _, v := range standbyHostList {
		this.StandbyHostName = append(this.StandbyHostName, &BucketOriginStandbyHost{
			Index:    v.Index,
			HostName: v.Standby,
		})
		this.StandbyHostName_N = append(this.StandbyHostName_N, v.Standby)
	}
	sort.Sort(privateStandbyHostList)
	for _, v := range privateStandbyHostList {
		this.PrivateStandbyHost_N = append(this.PrivateStandbyHost_N, v.PrivateStandby)
	}

	return nil
}

type SortStandby struct {
	Index          int64
	Standby        string
	PrivateStandby *BucketOriginPrivateHost
}
type SortStandbyList []SortStandby

func (s SortStandbyList) Len() int {
	return len(s)
}
func (s SortStandbyList) Less(i, j int) bool {
	return s[i].Index < s[j].Index
}
func (s SortStandbyList) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s *BucketService) PutOrigin(ctx context.Context, opt *BucketPutOriginOptions) (*Response, error) {
	sendOpt := &sendOptions{
		baseURL: s.client.BaseURL.BucketURL,
		uri:     "/?origin",
		method:  http.MethodPut,
		body:    opt,
	}
	resp, err := s.client.doRetry(ctx, sendOpt)
	return resp, err
}

func (s *BucketService) GetOrigin(ctx context.Context) (*BucketGetOriginResult, *Response, error) {
	var res BucketGetOriginResult
	sendOpt := &sendOptions{
		baseURL: s.client.BaseURL.BucketURL,
		uri:     "/?origin",
		method:  http.MethodGet,
		result:  &res,
	}
	resp, err := s.client.doRetry(ctx, sendOpt)
	return &res, resp, err
}

func (s *BucketService) DeleteOrigin(ctx context.Context) (*Response, error) {
	sendOpt := &sendOptions{
		baseURL: s.client.BaseURL.BucketURL,
		uri:     "/?origin",
		method:  http.MethodDelete,
	}
	resp, err := s.client.doRetry(ctx, sendOpt)
	return resp, err
}
