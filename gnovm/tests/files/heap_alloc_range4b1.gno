package main

func main() {
	var fns []func() int
	s := "hello"
	for i, _ := range s {
		f := func() int {
			return i
		}
		fns = append(fns, f)
	}
	for _, fn := range fns {
		println(fn())
	}
}

// Preprocessed:
// file{ package main; func main() { var fns<!VPBlock(1,0)> []func() .res.0 (const-type int); s<!VPBlock(1,1)> := (const ("hello" string)); for i<!~VPBlock(1,0)>, _<VPBlock(0,0)> := range s<VPBlock(2,1)> { f<!VPBlock(1,1)> := func func() .res.0 (const-type int){ return i<~VPBlock(1,1)> }<i<()~VPBlock(1,0)>>; fns<VPBlock(2,0)> = (const (append func([]func() int, ...func() int) []func() int))(fns<VPBlock(2,0)>, f<VPBlock(1,1)>) }; for _<VPBlock(0,0)>, fn<!VPBlock(1,0)> := range fns<VPBlock(2,0)> { (const (println func(...interface {})))(fn<VPBlock(1,0)>()) } } }

// Output:
// 0
// 1
// 2
// 3
// 4
