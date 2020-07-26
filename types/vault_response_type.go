package types

// VaultResponseType holds the response from the health check request
type VaultResponseType struct {
	Type         string `json:"type"`
	Initialized  bool   `json:"initialized"`
	Sealed       bool   `json:"sealed"`
	T            int    `json:"t"`
	N            int    `json:"n"`
	Progress     int    `json:"progress"`
	Nonce        string `json:"nonce"`
	Version      string `json:"version"`
	Migration    bool   `json:"migration"`
	ClusterName  string `json:"cluster_name"`
	ClusterID    string `json:"cluster_id"`
	RecoverySeal bool   `json:"recovery_seal"`
	StorageType  string `json:"storage_type"`
}
