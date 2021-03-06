package caffe

import (
	"strings"

	"github.com/k0kubun/pp"
	"github.com/rai-project/caffe"
	"github.com/rai-project/dllayer"
	"github.com/rai-project/dllayer/layer"

	toposort "github.com/philopon/go-toposort"
)

// See http://caffe.berkeleyvision.org/tutorial/layers.html

func kernelSizeOf(v uint32, size []uint32) uint32 {
	if v != 0 {
		return v
	}
	if v == 0 && len(size) == 0 {
		return 0
	}
	return size[0]
}

func kernelSizeOfPtr(v uint32, size *uint32) uint32 {
	if v != 0 {
		return v
	}
	if v == 0 && size == nil {
		return 0
	}
	return *size
}

func padSizeOf(v uint32, size []uint32) uint32 {
	if v != 0 {
		return v
	}
	if v == 0 && len(size) == 0 {
		return 0
	}
	return size[0]
}

func padSizeOfPtr(v uint32, size *uint32) uint32 {
	if v != 0 {
		return v
	}
	if v == 0 && size == nil {
		return 0
	}
	return *size
}

func strideSizeOf(v uint32, size []uint32) uint32 {
	if v != 0 {
		return v
	}
	if v == 0 && len(size) == 0 {
		return 0
	}
	return size[0]
}

func strideSizeOfPtr(v uint32, size *uint32) uint32 {
	if v != 0 {
		return v
	}
	if v == 0 && size == nil {
		return 0
	}
	return *size
}

func zeroIfNilUint32(v *uint32) uint32 {
	if v == nil {
		return 0
	}
	return *v
}

func orSlice(slc []uint32, v uint32) uint32 {
	if len(slc) == 0 {
		return v
	}
	return slc[0]
}

func orPtr(ptr *uint32, v uint32) uint32 {
	if ptr != nil {
		return *ptr
	}
	return v
}

func zeroToOne(v uint32) uint32 {
	if v == 0 {
		return 1
	}
	return v
}

func (c Caffe) Information() (dllayer.FlopsInformation, dllayer.MemoryInformation) {
	infos := c.LayerInformations()
	flops := dllayer.FlopsInformation{}
	memory := dllayer.MemoryInformation{}
	for _, info := range infos {
		flops = flops.Add(info.Flops())
		memory = memory.Add(info.Memory())
	}
	return flops, memory
}

func (c Caffe) FlopsInformation() dllayer.FlopsInformation {
	flops, _ := c.Information()
	return flops
}

func (c Caffe) MemoryInformation() dllayer.MemoryInformation {
	_, mem := c.Information()
	return mem
}

func (c Caffe) layerInformations(inputDimensions []int64) []dllayer.LayerInfo {

	graph := toposort.NewGraph(len(c.Layer))
	for _, lyr := range c.Layer {
		graph.AddNode(lyr.Name)
	}
	for _, lyr := range c.Layer {
		key := lyr.Name
		for _, bottom := range lyr.Bottom {
			graph.AddEdge(bottom, key)
		}
	}

	topSort, ok := graph.Toposort()
	if !ok {
		log.Error("failed to perform topological sort on graph")
		return nil
	}

	res := []dllayer.LayerInfo{}
	infos := map[string]dllayer.LayerInfo{}
	for _, name := range topSort {
		lyr := c.layers[name]
		layer := c.mkLayer(lyr)
		if layer == nil {
			pp.Println("failed to create ", name)
			return nil
		}
		if len(lyr.Bottom) == 0 {
			// nothing .. this is the root layer
		} else if len(lyr.Bottom) == 1 {
			bot := lyr.Bottom[0]
			if info, ok := infos[bot]; ok {
				inputDimensions = info.OutputDimensions()
			}
		} else if strings.ToLower(lyr.Type) != "concat" && strings.ToLower(lyr.Type) != "eltwise" {
			log.WithField("layer_type", lyr.Type).
				WithField("layer", lyr).
				Error("unhandeled to input dimension computation")
		}

		info := layer.LayerInformation(inputDimensions)
		c.layerInformation[lyr.Name] = info
		infos[name] = info
		res = append(res, info)
	}
	return res
}

func (c Caffe) layerInformationsV1(inputDimensions []int64) []dllayer.LayerInfo {

	graph := toposort.NewGraph(len(c.Layers))
	for _, lyr := range c.Layers {
		graph.AddNode(lyr.Name)
	}
	for _, lyr := range c.Layers {
		key := lyr.Name
		for _, bottom := range lyr.Bottom {
			graph.AddEdge(bottom, key)
		}
	}

	topSort, ok := graph.Toposort()
	if !ok {
		log.Error("failed to perform topological sort on graph")
		return nil
	}

	res := []dllayer.LayerInfo{}
	infos := map[string]dllayer.LayerInfo{}
	for _, name := range topSort {
		lyr := c.v1layers[name]
		layer := c.mkv1Layer(lyr)
		if layer == nil {
			pp.Println("failed to create ", name)
			return nil
		}
		if len(lyr.Bottom) == 0 {
			// nothing .. this is the root layer
		} else if len(lyr.Bottom) == 1 {
			bot := lyr.Bottom[0]
			if info, ok := infos[bot]; ok {
				inputDimensions = info.OutputDimensions()
			}
		} else if strings.ToLower(lyr.Type.String()) != "concat" {
			log.WithField("layer_type", lyr.Type).
				WithField("layer", lyr).
				Error("unhandeled to input dimension computation")
		}
		info := layer.LayerInformation(inputDimensions)
		c.layerInformation[lyr.Name] = info
		infos[name] = info
		res = append(res, info)
	}
	return res
}

func (c Caffe) LayerInformations() []dllayer.LayerInfo {

	var inputDimensions []int64
	if len(c.InputDim) != 0 {
		inputDimensions = dllayer.ToInt64Slice(c.InputDim)
	} else if len(c.InputShape) != 0 &&
		c.InputShape[0] != nil &&
		len(c.InputShape[0].Dim) != 0 {
		inputDimensions = dllayer.ToInt64Slice(c.InputShape[0].Dim)
	}

	if inputDimensions != nil {
		inputDimensions[0] = 1
	}
	if len(c.layers) != 0 {
		return c.layerInformations(inputDimensions)
	}

	if len(c.v1layers) != 0 {
		return c.layerInformationsV1(inputDimensions)
	}

	return nil
}

func (c Caffe) getParentsInfo(lyr *caffe.LayerParameter) []dllayer.LayerInfo {
	parentsInfo := []dllayer.LayerInfo{}
	for _, name := range lyr.Bottom {
		info, ok := c.layerInformation[name]
		if !ok {
			log.WithField("parent_name", name).WithField("layer_name", lyr.Name).Error("cannot find parent of concat layer")
		} else {
			parentsInfo = append(parentsInfo, info)
		}
	}

	return parentsInfo
}

func (c Caffe) mkLayer(lyr *caffe.LayerParameter) dllayer.Layer {
	var layer dllayer.Layer
	layerType := strings.ToLower(lyr.Type)
	switch layerType {
	case "input":
		layer = mkInput(lyr.InputParam)
	case "data":
		layer = mkData(lyr.InputParam)
	case "convolution":
		layer = mkConv(lyr.ConvolutionParam)
	case "relu":
		layer = mkReLU(lyr.ReluParam)
	case "dropout":
		layer = mkDropout(lyr.DropoutParam)
	case "innerproduct", "inner_product":
		layer = mkInnerProduct(lyr.InnerProductParam)
	case "pooling":
		layer = mkPooling(lyr.PoolingParam)
	case "batchnorm", "bn":
		layer = mkBatchNorm(lyr.BatchNormParam)
	case "lrn":
		layer = mkLRN(lyr.LrnParam)
	case "normalize":
	case "concat":
		parentsInfo := c.getParentsInfo(lyr)
		layer = mkConcat(parentsInfo, lyr.ConcatParam)
	case "eltwise":
		parentsInfo := c.getParentsInfo(lyr)
		layer = mkElementWise(parentsInfo, lyr.EltwiseParam)
	case "softmax", "softmaxwithloss", "softmax_loss":
		layer = mkSoftMax(lyr.SoftmaxParam)
	case "flatten":
		pp.Println("unhandeled", layerType)
	case "power":
		pp.Println("unhandeled", layerType)
	case "deconvolution":
		pp.Println("unhandeled", layerType)
	case "crop":
		pp.Println("unhandeled", layerType)
	case "scale":
		layer = mkScale(lyr.ScaleParam)
	case "implicit":
		pp.Println("unhandeled", layerType)
	case "accuracy":
		pp.Println("unhandeled", layerType)
	case "permute":
	default:
		pp.Println("unhandeled", layerType)

	}
	if layer == nil {
		pp.Println(lyr)
		return nil
	}
	layer.SetName(lyr.Name)

	return layer
}

func (c Caffe) getParentsInfoV1(lyr *caffe.V1LayerParameter) []dllayer.LayerInfo {
	parentsInfo := []dllayer.LayerInfo{}
	for _, name := range lyr.Bottom {
		info, ok := c.layerInformation[name]
		if !ok {
			log.WithField("parent_name", name).WithField("layer_name", lyr.Name).Error("cannot find parent of concat layer")
		}
		if ok {
			parentsInfo = append(parentsInfo, info)
		}
	}

	return parentsInfo
}

func (c Caffe) mkv1Layer(lyr *caffe.V1LayerParameter) dllayer.Layer {
	var layer dllayer.Layer
	layerType := strings.ToLower(lyr.Type.String())
	switch layerType {
	case "data":
		data := lyr.DataParam
		pp.Println(data)
	case "convolution":
		layer = mkConv(lyr.ConvolutionParam)
	case "relu":
		layer = mkReLU(lyr.ReluParam)
	case "dropout":
		layer = mkDropout(lyr.DropoutParam)
	case "innerproduct", "inner_product":
		layer = mkInnerProduct(lyr.InnerProductParam)
	case "pooling":
		layer = mkPooling(lyr.PoolingParam)
	case "lrn":
		layer = mkLRN(lyr.LrnParam)
	case "normalize":
	case "concat":
		parentsInfo := c.getParentsInfoV1(lyr)
		layer = mkConcat(parentsInfo, lyr.ConcatParam)
	case "eltwise":
		parentsInfo := c.getParentsInfoV1(lyr)
		layer = mkElementWise(parentsInfo, lyr.EltwiseParam)
	case "softmax", "softmaxwithloss", "softmax_loss":
		layer = mkSoftMax(lyr.SoftmaxParam)
	case "flatten":
		pp.Println("unhandeled", layerType)
	case "power":
		pp.Println("unhandeled", layerType)
	case "deconvolution":
		pp.Println("unhandeled", layerType)
	case "crop":
		pp.Println("unhandeled", layerType)
	case "implicit":
		pp.Println("unhandeled", layerType)
	case "accuracy":
		pp.Println("unhandeled", layerType)
	case "permute":
		pp.Println("unhandeled", layerType)
	default:
		pp.Println("unhandeled", layerType)
	}
	if layer == nil {
		pp.Println(lyr)
		return nil
	}
	layer.SetName(lyr.Name)

	return layer
}

func mkInput(param *caffe.InputParameter) dllayer.Layer {
	inputDimensions := dllayer.ToInt64Slice(param.Shape[0].Dim)
	return &layer.Input{
		N: inputDimensions[0],
		C: inputDimensions[1],
		W: inputDimensions[2],
		H: inputDimensions[3],
	}
}

func mkData(param *caffe.InputParameter) dllayer.Layer {
	inputDimensions := dllayer.ToInt64Slice(param.Shape[0].Dim)
	return &layer.Data{
		N: inputDimensions[0],
		C: inputDimensions[1],
		W: inputDimensions[2],
		H: inputDimensions[3],
	}
}

func mkConv(param *caffe.ConvolutionParameter) dllayer.Layer {
	return &layer.Convolution{
		NumOutput: param.NumOutput,
		PadH:      padSizeOf(zeroIfNilUint32(param.PadH), param.Pad),
		PadW:      padSizeOf(zeroIfNilUint32(param.PadW), param.Pad),
		KernelH:   kernelSizeOf(param.KernelH, param.KernelSize),
		KernelW:   kernelSizeOf(param.KernelW, param.KernelSize),
		StrideH:   zeroToOne(strideSizeOf(param.StrideH, param.Stride)),
		StrideW:   zeroToOne(strideSizeOf(param.StrideW, param.Stride)),
		Dilation:  orSlice(param.Dilation, 1),
		Group:     orPtr(param.Group, 1),
	}
}

func mkReLU(param *caffe.ReLUParameter) dllayer.Layer {
	return &layer.ReLU{}
}

func mkDropout(param *caffe.DropoutParameter) dllayer.Layer {
	return &layer.Dropout{}
}

func mkScale(param *caffe.ScaleParameter) dllayer.Layer {
	return &layer.Scale{}
}

func mkSoftMax(param *caffe.SoftmaxParameter) dllayer.Layer {
	return &layer.SoftMax{}
}

func mkBatchNorm(param *caffe.BatchNormParameter) dllayer.Layer {
	return &layer.BatchNorm{}
}

func mkLRN(param *caffe.LRNParameter) dllayer.Layer {
	size := uint32(1)
	if param != nil && param.LocalSize != nil && *param.LocalSize != 0 {
		size = *param.LocalSize
	}
	region := "ACROSS_CHANNELS"
	if param != nil && param.NormRegion != nil && *param.NormRegion != 0 {
		region = "WITHIN_CHANNEL"
	}

	return &layer.LRN{
		Region: region,
		Size:   size,
	}
}

func mkPooling(param *caffe.PoolingParameter) dllayer.Layer {
	operator := "max"
	if param.Pool != nil && *param.Pool != 0 {
		operator = param.Pool.String()
	}
	global := false
	if param.GlobalPooling != nil {
		global = *param.GlobalPooling
	}
	return &layer.Pooling{
		Operator: strings.ToUpper(operator),
		PadH:     padSizeOfPtr(zeroIfNilUint32(param.PadH), param.Pad),
		PadW:     padSizeOfPtr(zeroIfNilUint32(param.PadW), param.Pad),
		KernelH:  kernelSizeOfPtr(param.KernelH, &param.KernelSize),
		KernelW:  kernelSizeOfPtr(param.KernelW, &param.KernelSize),
		StrideH:  zeroToOne(strideSizeOfPtr(param.StrideH, param.Stride)),
		StrideW:  zeroToOne(strideSizeOfPtr(param.StrideW, param.Stride)),
		Global:   global,
	}
}

func mkInnerProduct(param *caffe.InnerProductParameter) dllayer.Layer {
	return &layer.InnerProduct{
		NumOutput: param.NumOutput,
	}
}

func mkConcat(parentsInfo []dllayer.LayerInfo, param *caffe.ConcatParameter) dllayer.Layer {
	return &layer.Concat{
		ParentsInformation: parentsInfo,
	}
}

func mkElementWise(parentsInfo []dllayer.LayerInfo, param *caffe.EltwiseParameter) dllayer.Layer {
	op := "SUM"
	if param != nil && param.Operation != nil && param.Operation.String() != "" {
		op = param.Operation.String()
	}
	return &layer.ElementWise{
		Operation:          op,
		ParentsInformation: parentsInfo,
	}
}
