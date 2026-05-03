package aimini

import "encoding/json"

// UnmarshalJSON accepts the contract shape and the older deployed queue.add
// shape. Callers always receive QueueAddResponse.
func (r *QueueAddResponse) UnmarshalJSON(data []byte) error {
	type addResponse QueueAddResponse
	var current addResponse
	if err := json.Unmarshal(data, &current); err != nil {
		return err
	}
	if current.Item.ID != "" || current.QueueSize != 0 {
		*r = QueueAddResponse(current)
		return nil
	}

	var legacy QueueItem
	if err := json.Unmarshal(data, &legacy); err != nil {
		return err
	}
	r.Item = legacy
	return nil
}
