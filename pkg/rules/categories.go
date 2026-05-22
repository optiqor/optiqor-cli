package rules

// Detector → Category bindings live here so the classification is
// auditable in one place. Adding a new detector requires:
//
//  1. Implementing the Detector interface in its own file.
//  2. Registering it in [All].
//  3. Adding its Category() method below.
//
// The build will fail if step 3 is skipped because [Detector] requires
// Category(); that is intentional — every finding must declare which
// product surface it serves.

// ---- Cost / efficiency ------------------------------------------------

func (cpuOverprovisioned) Category() Category         { return CategoryCost }
func (memoryOverprovisioned) Category() Category      { return CategoryCost }
func (cpuLimitFarAboveRequest) Category() Category    { return CategoryCost }
func (memoryLimitFarAboveRequest) Category() Category { return CategoryCost }
func (oversizedCPULimit) Category() Category          { return CategoryCost }
func (oversizedMemoryLimit) Category() Category       { return CategoryCost }
func (replicasTooHigh) Category() Category            { return CategoryCost }
func (excessiveReplicaCount) Category() Category      { return CategoryCost }
func (unboundedImageTag) Category() Category          { return CategoryCost }
func (cpuWithoutMemoryRequest) Category() Category    { return CategoryCost }
func (memoryWithoutCPURequest) Category() Category    { return CategoryCost }
func (cpuRequestEqualsLimit) Category() Category      { return CategoryCost }
func (memoryRequestEqualsLimit) Category() Category   { return CategoryCost }
func (tinyCPURequest) Category() Category             { return CategoryCost }
func (tinyMemoryRequest) Category() Category          { return CategoryCost }
func (idleWorkload) Category() Category               { return CategoryCost }

// ---- Security --------------------------------------------------------

func (missingCPULimit) Category() Category              { return CategorySecurity }
func (missingMemoryLimit) Category() Category           { return CategorySecurity }
func (imagePinnedLatest) Category() Category            { return CategorySecurity }
func (runAsRoot) Category() Category                    { return CategorySecurity }
func (runsAsUIDZero) Category() Category                { return CategorySecurity }
func (privilegedContainer) Category() Category          { return CategorySecurity }
func (hostNetwork) Category() Category                  { return CategorySecurity }
func (hostPID) Category() Category                      { return CategorySecurity }
func (hostIPC) Category() Category                      { return CategorySecurity }
func (readOnlyRootFSMissing) Category() Category        { return CategorySecurity }
func (allowPrivilegeEscalation) Category() Category     { return CategorySecurity }
func (hostPathVolume) Category() Category               { return CategorySecurity }
func (dangerousCapabilityAdded) Category() Category     { return CategorySecurity }
func (capabilitiesNotDroppedAll) Category() Category    { return CategorySecurity }
func (serviceAccountTokenAutomount) Category() Category { return CategorySecurity }
