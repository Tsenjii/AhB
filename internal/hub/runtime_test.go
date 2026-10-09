package hub

import (
 "os"
 "path/filepath"
 "testing"
)

func TestMemInfoSnapshot(t *testing.T) {
 path:=filepath.Join(t.TempDir(),"meminfo")
 if err:=os.WriteFile(path,[]byte("MemTotal:      524288 kB\nMemFree: 131072 kB\nMemAvailable: 262144 kB\n"),0600);err!=nil {t.Fatal(err)}
 s:=memInfoSnapshot(path)
 if s.total!=536870912||s.available!=268435456||s.used!=268435456||s.source!="host" {t.Fatalf("%+v",s)}
}

func TestCgroupMemoryQuota(t *testing.T) {
 dir:=t.TempDir()
 max:=filepath.Join(dir,"memory.max")
 current:=filepath.Join(dir,"memory.current")
 if err:=os.WriteFile(max,[]byte("536870912\n"),0600);err!=nil {t.Fatal(err)}
 if err:=os.WriteFile(current,[]byte("402653184\n"),0600);err!=nil {t.Fatal(err)}
 s:=cgroupSnapshot(max,current)
 if s.total!=536870912||s.used!=402653184||s.available!=134217728||s.source!="cgroup_v2" {t.Fatalf("%+v",s)}
 if err:=os.WriteFile(max,[]byte("max\n"),0600);err!=nil {t.Fatal(err)}
 if s:=cgroupSnapshot(max,current);s.total!=0 {t.Fatal("unbounded cgroup should not override host memory")}
}
