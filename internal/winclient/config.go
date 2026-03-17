package winclient

import (
	"fmt"
	"strconv"
	"strings"
)

type Operation string
const (
	OperationVolume Operation = "volume"
	OperationGetAttr Operation = "getattr"
	OperationReadDir Operation = "readdir"
	OperationRead Operation = "read"
	OperationMaterialize Operation = "materialize"
)

type Config struct {
	Addr string
	Token string
	ClientInstanceID string
	LeaseSeconds uint32
	MountPoint string
	VolumePrefix string
	Path string
	LocalPath string
	Offset int64
	Length uint32
	MaxEntries uint32
}
func DefaultConfig() Config { return Config{Addr:"127.0.0.1:17890",Token:"devmount-dev-token",ClientInstanceID:"win32-test-ui",LeaseSeconds:30,MountPoint:"M:",VolumePrefix:"devmount",Path:"/",LocalPath:"devmount-local",Offset:0,Length:64,MaxEntries:32} }
func Operations() []Operation { return []Operation{OperationVolume,OperationGetAttr,OperationReadDir,OperationRead,OperationMaterialize} }
func (c Config) Normalized() Config { d:=DefaultConfig(); if strings.TrimSpace(c.Addr)==""{c.Addr=d.Addr}; if strings.TrimSpace(c.Token)==""{c.Token=d.Token}; if strings.TrimSpace(c.ClientInstanceID)==""{c.ClientInstanceID=d.ClientInstanceID}; if c.LeaseSeconds==0{c.LeaseSeconds=d.LeaseSeconds}; if strings.TrimSpace(c.MountPoint)==""{c.MountPoint=d.MountPoint}; if strings.TrimSpace(c.VolumePrefix)==""{c.VolumePrefix=d.VolumePrefix}; if strings.TrimSpace(c.Path)==""{c.Path=d.Path}; if !strings.HasPrefix(c.Path,"/"){c.Path="/"+c.Path}; if strings.TrimSpace(c.LocalPath)==""{c.LocalPath=d.LocalPath}; if c.Length==0{c.Length=d.Length}; if c.MaxEntries==0{c.MaxEntries=d.MaxEntries}; return c }
func (c Config) Validate(op Operation) error { c=c.Normalized(); if strings.TrimSpace(c.Addr)==""{return fmt.Errorf("server address is required")}; if strings.TrimSpace(c.Token)==""{return fmt.Errorf("token is required")}; if strings.TrimSpace(c.ClientInstanceID)==""{return fmt.Errorf("client instance id is required")}; if c.LeaseSeconds==0{return fmt.Errorf("lease seconds must be greater than 0")}; if strings.TrimSpace(c.VolumePrefix)==""{return fmt.Errorf("volume prefix is required")}; if strings.TrimSpace(c.Path)==""{return fmt.Errorf("path is required")}; if !strings.HasPrefix(c.Path,"/"){return fmt.Errorf("path must start with '/'")}; if op==OperationMaterialize && strings.TrimSpace(c.LocalPath)==""{return fmt.Errorf("local path is required for materialize")}; switch op{case OperationVolume,OperationGetAttr,OperationReadDir,OperationRead,OperationMaterialize: default:return fmt.Errorf("unsupported operation %q",op)}; return nil }
func BuildCLIPreview(config Config, op Operation) string { config=config.Normalized(); args:=[]string{"devmount-winfsp.exe","-addr",quoteIfNeeded(config.Addr),"-token",quoteIfNeeded(config.Token),"-client-instance",quoteIfNeeded(config.ClientInstanceID),"-op",string(op),"-path",quoteIfNeeded(config.Path),"-mount-point",quoteIfNeeded(config.MountPoint),"-volume-prefix",quoteIfNeeded(config.VolumePrefix)}; switch op{case OperationRead: args=append(args,"-offset",strconv.FormatInt(config.Offset,10),"-length",strconv.FormatUint(uint64(config.Length),10)); case OperationReadDir: args=append(args,"-max-entries",strconv.FormatUint(uint64(config.MaxEntries),10)); case OperationMaterialize: args=append(args,"-local-path",quoteIfNeeded(config.LocalPath),"-length",strconv.FormatUint(uint64(config.Length),10),"-max-entries",strconv.FormatUint(uint64(config.MaxEntries),10))}; return strings.Join(args," ") }
func quoteIfNeeded(s string) string { if s==""{return `""`}; if strings.IndexFunc(s,func(r rune) bool { return r==' '||r=='\t'||r=='"' })==-1{return s}; escaped:=strings.ReplaceAll(s,`"`,`\\"`); return `"`+escaped+`"` }
