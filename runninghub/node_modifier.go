package runninghub

type NodeModifier struct {
	modifications []WorkflowNodeInfo
}

func ModifyNodes() *NodeModifier {
	return &NodeModifier{}
}

func NewNodeModifier() *NodeModifier {
	return ModifyNodes()
}

func (m *NodeModifier) Set(nodeID, fieldName string, fieldValue any) *NodeModifier {
	if m == nil {
		m = &NodeModifier{}
	}
	m.modifications = append(m.modifications, WorkflowNodeInfo{
		NodeID:     nodeID,
		FieldName:  fieldName,
		FieldValue: fieldValue,
	})
	return m
}

func (m *NodeModifier) Text(nodeID, text string) *NodeModifier {
	return m.Set(nodeID, "text", text)
}

func (m *NodeModifier) NegativeText(nodeID, text string) *NodeModifier {
	return m.Set(nodeID, "text", text)
}

func (m *NodeModifier) Seed(nodeID string, seed int) *NodeModifier {
	return m.Set(nodeID, "seed", seed)
}

func (m *NodeModifier) Steps(nodeID string, steps int) *NodeModifier {
	return m.Set(nodeID, "steps", steps)
}

func (m *NodeModifier) CFG(nodeID string, cfg float64) *NodeModifier {
	return m.Set(nodeID, "cfg", cfg)
}

func (m *NodeModifier) Size(nodeID string, width, height int, batchSize ...int) *NodeModifier {
	m.Set(nodeID, "width", width)
	m.Set(nodeID, "height", height)
	if len(batchSize) > 0 {
		m.Set(nodeID, "batch_size", batchSize[0])
	}
	return m
}

func (m *NodeModifier) Sampler(nodeID, samplerName string) *NodeModifier {
	return m.Set(nodeID, "sampler_name", samplerName)
}

func (m *NodeModifier) Scheduler(nodeID, scheduler string) *NodeModifier {
	return m.Set(nodeID, "scheduler", scheduler)
}

func (m *NodeModifier) Denoise(nodeID string, denoise float64) *NodeModifier {
	return m.Set(nodeID, "denoise", denoise)
}

func (m *NodeModifier) Image(nodeID, fileName string) *NodeModifier {
	return m.Set(nodeID, "image", fileName)
}

func (m *NodeModifier) Video(nodeID, fileName string) *NodeModifier {
	return m.Set(nodeID, "video", fileName)
}

func (m *NodeModifier) Audio(nodeID, fileName string) *NodeModifier {
	return m.Set(nodeID, "audio", fileName)
}

func (m *NodeModifier) Lora(nodeID, loraFileName string) *NodeModifier {
	return m.Set(nodeID, "lora_name", loraFileName)
}

func (m *NodeModifier) LoraStrength(nodeID string, strength float64) *NodeModifier {
	return m.Set(nodeID, "strength_model", strength)
}

func (m *NodeModifier) Checkpoint(nodeID, ckptName string) *NodeModifier {
	return m.Set(nodeID, "ckpt_name", ckptName)
}

func (m *NodeModifier) AddMany(modifications []WorkflowNodeInfo) *NodeModifier {
	if m == nil {
		m = &NodeModifier{}
	}
	m.modifications = append(m.modifications, modifications...)
	return m
}

func (m *NodeModifier) ToWorkflowNodeInfoList() []WorkflowNodeInfo {
	if m == nil || len(m.modifications) == 0 {
		return nil
	}
	out := make([]WorkflowNodeInfo, len(m.modifications))
	copy(out, m.modifications)
	return out
}

func (m *NodeModifier) ToAIAppNodeInfoList() []AIAppNodeInfo {
	if m == nil || len(m.modifications) == 0 {
		return nil
	}
	out := make([]AIAppNodeInfo, 0, len(m.modifications))
	for _, item := range m.modifications {
		out = append(out, AIAppNodeInfo{
			NodeID:     item.NodeID,
			FieldName:  item.FieldName,
			FieldValue: item.FieldValue,
		})
	}
	return out
}

func (m *NodeModifier) ToList() []NodeInput {
	return m.ToWorkflowNodeInfoList()
}