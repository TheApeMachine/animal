package a2a

import "maps"

/*
Clone copies a part and its mutable data map.
*/
func (part Part) Clone() Part {
	clone := part

	if part.Data != nil {
		clone.Data = make(map[string]any, len(part.Data))
		maps.Copy(clone.Data, part.Data)
	}

	return clone
}

/*
Clone copies a message and its mutable collections.
*/
func (message Message) Clone() Message {
	clone := message
	clone.Parts = make([]Part, 0, len(message.Parts))

	for _, part := range message.Parts {
		clone.Parts = append(clone.Parts, part.Clone())
	}

	clone.ReferenceTaskIDs = append([]string(nil), message.ReferenceTaskIDs...)
	clone.Metadata = cloneMetadata(message.Metadata)

	return clone
}

/*
Clone copies an artifact and its mutable collections.
*/
func (artifact Artifact) Clone() Artifact {
	clone := artifact
	clone.Parts = make([]Part, 0, len(artifact.Parts))

	for _, part := range artifact.Parts {
		clone.Parts = append(clone.Parts, part.Clone())
	}

	clone.Metadata = cloneMetadata(artifact.Metadata)

	return clone
}

/*
Clone copies a task status and its optional message.
*/
func (status TaskStatus) Clone() TaskStatus {
	clone := status

	if status.Message != nil {
		message := status.Message.Clone()
		clone.Message = &message
	}

	return clone
}

/*
Clone copies a task and its mutable collections.
*/
func (task Task) Clone() Task {
	clone := task
	clone.Status = task.Status.Clone()
	clone.Artifacts = make([]Artifact, 0, len(task.Artifacts))

	for _, artifact := range task.Artifacts {
		clone.Artifacts = append(clone.Artifacts, artifact.Clone())
	}

	clone.History = make([]Message, 0, len(task.History))

	for _, message := range task.History {
		clone.History = append(clone.History, message.Clone())
	}

	clone.Metadata = cloneMetadata(task.Metadata)

	return clone
}

func cloneMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return nil
	}

	clone := make(map[string]any, len(metadata))
	maps.Copy(clone, metadata)

	return clone
}
