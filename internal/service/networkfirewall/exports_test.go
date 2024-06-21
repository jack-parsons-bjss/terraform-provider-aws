// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package networkfirewall

// Exports for use in tests only.
var (
	ResourceFirewall                   = resourceFirewall
	ResourceFirewallPolicy             = resourceFirewallPolicy
	ResourceLoggingConfiguration       = resourceLoggingConfiguration
	ResourceResourcePolicy             = resourceResourcePolicy
	ResourceTLSInspectionConfiguration = newTLSInspectionConfigurationResource

	FindFirewallByARN                   = findFirewallByARN
	FindFirewallPolicyByARN             = findFirewallPolicyByARN
	FindLoggingConfigurationByARN       = findLoggingConfigurationByARN
	FindResourcePolicyByARN             = findResourcePolicyByARN
	FindTLSInspectionConfigurationByARN = findTLSInspectionConfigurationByARN
)
