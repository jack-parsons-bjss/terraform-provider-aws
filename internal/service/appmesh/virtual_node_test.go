package appmesh_test

import (
	"fmt"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/acmpca"
	"github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/go-multierror"
	sdkacctest "github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
)

func init() {
	resource.AddTestSweepers("aws_appmesh_virtual_node", &resource.Sweeper{
		Name: "aws_appmesh_virtual_node",
		F:    testSweepAppmeshVirtualNodes,
	})
}

func testSweepAppmeshVirtualNodes(region string) error {
	client, err := acctest.SharedRegionalSweeperClient(region)
	if err != nil {
		return fmt.Errorf("error getting client: %w", err)
	}
	conn := client.(*conns.AWSClient).AppMeshConn

	var sweeperErrs *multierror.Error

	err = conn.ListMeshesPages(&appmesh.ListMeshesInput{}, func(page *appmesh.ListMeshesOutput, lastPage bool) bool {
		if page == nil {
			return !lastPage
		}

		for _, mesh := range page.Meshes {
			listVirtualNodesInput := &appmesh.ListVirtualNodesInput{
				MeshName: mesh.MeshName,
			}
			meshName := aws.StringValue(mesh.MeshName)

			err := conn.ListVirtualNodesPages(listVirtualNodesInput, func(page *appmesh.ListVirtualNodesOutput, lastPage bool) bool {
				if page == nil {
					return !lastPage
				}

				for _, virtualNode := range page.VirtualNodes {
					input := &appmesh.DeleteVirtualNodeInput{
						MeshName:        mesh.MeshName,
						VirtualNodeName: virtualNode.VirtualNodeName,
					}
					virtualNodeName := aws.StringValue(virtualNode.VirtualNodeName)

					log.Printf("[INFO] Deleting Appmesh Mesh (%s) Virtual Node: %s", meshName, virtualNodeName)
					_, err := conn.DeleteVirtualNode(input)

					if err != nil {
						sweeperErr := fmt.Errorf("error deleting Appmesh Mesh (%s) Virtual Node (%s): %w", meshName, virtualNodeName, err)
						log.Printf("[ERROR] %s", sweeperErr)
						sweeperErrs = multierror.Append(sweeperErrs, sweeperErr)
						continue
					}
				}

				return !lastPage
			})

			if err != nil {
				log.Printf("[ERROR] Error retrieving Appmesh Mesh (%s) Virtual Nodes: %s", meshName, err)
			}
		}

		return !lastPage
	})
	if err != nil {
		if acctest.SkipSweepError(err) {
			log.Printf("[WARN] Skipping Appmesh Virtual Node sweep for %s: %s", region, err)
			return nil
		}
		return fmt.Errorf("error retrieving Appmesh Virtual Nodes: %w", err)
	}

	return sweeperErrs.ErrorOrNil()
}

func testAccAwsAppmeshVirtualNode_basic(t *testing.T) {
	var vn appmesh.VirtualNodeData
	resourceName := "aws_appmesh_virtual_node.test"
	meshName := sdkacctest.RandomWithPrefix("tf-acc-test")
	vnName := sdkacctest.RandomWithPrefix("tf-acc-test")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); acctest.PreCheckPartitionHasService(appmesh.EndpointsID, t) },
		ErrorCheck:   acctest.ErrorCheck(t, appmesh.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAppmeshVirtualNodeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAppmeshVirtualNodeConfig_basic(meshName, vnName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAppmeshVirtualNodeExists(resourceName, &vn),
					resource.TestCheckResourceAttr(resourceName, "name", vnName),
					resource.TestCheckResourceAttr(resourceName, "mesh_name", meshName),
					acctest.CheckResourceAttrAccountID(resourceName, "mesh_owner"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.logging.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.#", "0"),
					resource.TestCheckResourceAttrSet(resourceName, "created_date"),
					resource.TestCheckResourceAttrSet(resourceName, "last_updated_date"),
					acctest.CheckResourceAttrAccountID(resourceName, "resource_owner"),
					acctest.CheckResourceAttrRegionalARN(resourceName, "arn", "appmesh", fmt.Sprintf("mesh/%s/virtualNode/%s", meshName, vnName)),
				),
			},
			{
				ResourceName:      resourceName,
				ImportStateId:     fmt.Sprintf("%s/%s", meshName, vnName),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccAwsAppmeshVirtualNode_disappears(t *testing.T) {
	var vn appmesh.VirtualNodeData
	resourceName := "aws_appmesh_virtual_node.test"
	meshName := sdkacctest.RandomWithPrefix("tf-acc-test")
	vnName := sdkacctest.RandomWithPrefix("tf-acc-test")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); acctest.PreCheckPartitionHasService(appmesh.EndpointsID, t) },
		ErrorCheck:   acctest.ErrorCheck(t, appmesh.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAppmeshVirtualNodeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAppmeshVirtualNodeConfig_basic(meshName, vnName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAppmeshVirtualNodeExists(resourceName, &vn),
					acctest.CheckResourceDisappears(acctest.Provider, ResourceVirtualNode(), resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccAwsAppmeshVirtualNode_backendClientPolicyAcm(t *testing.T) {
	var vn appmesh.VirtualNodeData
	var ca acmpca.CertificateAuthority
	resourceName := "aws_appmesh_virtual_node.test"
	acmCAResourceName := "aws_acmpca_certificate_authority.test"

	meshName := sdkacctest.RandomWithPrefix("tf-acc-test")
	vnName := sdkacctest.RandomWithPrefix("tf-acc-test")
	domain := acctest.RandomDomainName()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); acctest.PreCheckPartitionHasService(appmesh.EndpointsID, t) },
		ErrorCheck:   acctest.ErrorCheck(t, appmesh.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAppmeshVirtualNodeDestroy,
		Steps: []resource.TestStep{
			// We need to create and activate the CA before issuing a certificate.
			{
				Config: testAccAppmeshVirtualNodeConfigRootCA(domain),
				Check: resource.ComposeTestCheckFunc(
					acctest.CheckACMPCACertificateAuthorityExists(acmCAResourceName, &ca),
					acctest.CheckACMPCACertificateAuthorityActivateCA(&ca),
				),
			},
			{
				Config: testAccAppmeshVirtualNodeConfig_backendClientPolicyAcm(meshName, vnName, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAppmeshVirtualNodeExists(resourceName, &vn),
					resource.TestCheckResourceAttr(resourceName, "name", vnName),
					resource.TestCheckResourceAttr(resourceName, "mesh_name", meshName),
					acctest.CheckResourceAttrAccountID(resourceName, "mesh_owner"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "spec.0.backend.*", map[string]string{
						"virtual_service.#":                                                                "1",
						"virtual_service.0.client_policy.#":                                                "1",
						"virtual_service.0.client_policy.0.tls.#":                                          "1",
						"virtual_service.0.client_policy.0.tls.0.certificate.#":                            "0",
						"virtual_service.0.client_policy.0.tls.0.enforce":                                  "true",
						"virtual_service.0.client_policy.0.tls.0.ports.#":                                  "1",
						"virtual_service.0.client_policy.0.tls.0.validation.#":                             "1",
						"virtual_service.0.client_policy.0.tls.0.validation.0.subject_alternative_names.#": "0",
						"virtual_service.0.client_policy.0.tls.0.validation.0.trust.#":                     "1",
						"virtual_service.0.client_policy.0.tls.0.validation.0.trust.0.acm.#":               "1",
						"virtual_service.0.client_policy.0.tls.0.validation.0.trust.0.file.#":              "0",
						"virtual_service.0.client_policy.0.tls.0.validation.0.trust.0.sds.#":               "0",
						"virtual_service.0.virtual_service_name":                                           "servicea.simpleapp.local",
					}),
					resource.TestCheckTypeSetElemAttr(resourceName, "spec.0.backend.*.virtual_service.0.client_policy.0.tls.0.ports.*", "8443"),
					resource.TestCheckTypeSetElemAttrPair(resourceName, "spec.0.backend.*.virtual_service.0.client_policy.0.tls.0.validation.0.trust.0.acm.0.certificate_authority_arns.*", acmCAResourceName, "arn"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.connection_pool.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.health_check.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.outlier_detection.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.0.port", "8080"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.0.protocol", "http"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.logging.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.dns.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.dns.0.hostname", "serviceb.simpleapp.local"),
					resource.TestCheckResourceAttrSet(resourceName, "created_date"),
					resource.TestCheckResourceAttrSet(resourceName, "last_updated_date"),
					acctest.CheckResourceAttrAccountID(resourceName, "resource_owner"),
					acctest.CheckResourceAttrRegionalARN(resourceName, "arn", "appmesh", fmt.Sprintf("mesh/%s/virtualNode/%s", meshName, vnName)),
				),
			},
			{
				ResourceName:      resourceName,
				ImportStateId:     fmt.Sprintf("%s/%s", meshName, vnName),
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAppmeshVirtualNodeConfig_backendClientPolicyAcm(meshName, vnName, domain),
				Check: resource.ComposeTestCheckFunc(
					// CA must be DISABLED for deletion.
					acctest.CheckACMPCACertificateAuthorityDisableCA(&ca),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccAwsAppmeshVirtualNode_backendClientPolicyFile(t *testing.T) {
	var vn appmesh.VirtualNodeData
	resourceName := "aws_appmesh_virtual_node.test"
	meshName := sdkacctest.RandomWithPrefix("tf-acc-test")
	vnName := sdkacctest.RandomWithPrefix("tf-acc-test")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); acctest.PreCheckPartitionHasService(appmesh.EndpointsID, t) },
		ErrorCheck:   acctest.ErrorCheck(t, appmesh.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAppmeshVirtualNodeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAppmeshVirtualNodeConfig_backendClientPolicyFile(meshName, vnName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAppmeshVirtualNodeExists(resourceName, &vn),
					resource.TestCheckResourceAttr(resourceName, "name", vnName),
					resource.TestCheckResourceAttr(resourceName, "mesh_name", meshName),
					acctest.CheckResourceAttrAccountID(resourceName, "mesh_owner"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "spec.0.backend.*", map[string]string{
						"virtual_service.#":                                                                     "1",
						"virtual_service.0.client_policy.#":                                                     "1",
						"virtual_service.0.client_policy.0.tls.#":                                               "1",
						"virtual_service.0.client_policy.0.tls.0.certificate.#":                                 "0",
						"virtual_service.0.client_policy.0.tls.0.enforce":                                       "true",
						"virtual_service.0.client_policy.0.tls.0.ports.#":                                       "1",
						"virtual_service.0.client_policy.0.tls.0.validation.#":                                  "1",
						"virtual_service.0.client_policy.0.tls.0.validation.0.subject_alternative_names.#":      "0",
						"virtual_service.0.client_policy.0.tls.0.validation.0.trust.#":                          "1",
						"virtual_service.0.client_policy.0.tls.0.validation.0.trust.0.acm.#":                    "0",
						"virtual_service.0.client_policy.0.tls.0.validation.0.trust.0.file.#":                   "1",
						"virtual_service.0.client_policy.0.tls.0.validation.0.trust.0.file.0.certificate_chain": "/cert_chain.pem",
						"virtual_service.0.client_policy.0.tls.0.validation.0.trust.0.sds.#":                    "0",
						"virtual_service.0.virtual_service_name":                                                "servicea.simpleapp.local",
					}),
					resource.TestCheckTypeSetElemAttr(resourceName, "spec.0.backend.*.virtual_service.0.client_policy.0.tls.0.ports.*", "8443"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.connection_pool.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.health_check.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.outlier_detection.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.0.port", "8080"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.0.protocol", "http"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.logging.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.dns.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.dns.0.hostname", "serviceb.simpleapp.local"),
					resource.TestCheckResourceAttrSet(resourceName, "created_date"),
					resource.TestCheckResourceAttrSet(resourceName, "last_updated_date"),
					acctest.CheckResourceAttrAccountID(resourceName, "resource_owner"),
					acctest.CheckResourceAttrRegionalARN(resourceName, "arn", "appmesh", fmt.Sprintf("mesh/%s/virtualNode/%s", meshName, vnName)),
				),
			},
			{
				Config: testAccAppmeshVirtualNodeConfig_backendClientPolicyFileUpdated(meshName, vnName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAppmeshVirtualNodeExists(resourceName, &vn),
					resource.TestCheckResourceAttr(resourceName, "name", vnName),
					resource.TestCheckResourceAttr(resourceName, "mesh_name", meshName),
					acctest.CheckResourceAttrAccountID(resourceName, "mesh_owner"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "spec.0.backend.*", map[string]string{
						"virtual_service.#":                                                                                "1",
						"virtual_service.0.client_policy.#":                                                                "1",
						"virtual_service.0.client_policy.0.tls.#":                                                          "1",
						"virtual_service.0.client_policy.0.tls.0.enforce":                                                  "true",
						"virtual_service.0.client_policy.0.tls.0.ports.#":                                                  "2",
						"virtual_service.0.client_policy.0.tls.0.validation.#":                                             "1",
						"virtual_service.0.client_policy.0.tls.0.validation.0.subject_alternative_names.#":                 "1",
						"virtual_service.0.client_policy.0.tls.0.validation.0.subject_alternative_names.0.match.#":         "1",
						"virtual_service.0.client_policy.0.tls.0.validation.0.subject_alternative_names.0.match.0.exact.#": "1",
						"virtual_service.0.client_policy.0.tls.0.validation.0.trust.#":                                     "1",
						"virtual_service.0.client_policy.0.tls.0.validation.0.trust.0.acm.#":                               "0",
						"virtual_service.0.client_policy.0.tls.0.validation.0.trust.0.file.#":                              "1",
						"virtual_service.0.client_policy.0.tls.0.validation.0.trust.0.file.0.certificate_chain":            "/etc/ssl/certs/cert_chain.pem",
						"virtual_service.0.virtual_service_name":                                                           "servicea.simpleapp.local",
					}),
					resource.TestCheckTypeSetElemAttr(resourceName, "spec.0.backend.*.virtual_service.0.client_policy.0.tls.0.ports.*", "443"),
					resource.TestCheckTypeSetElemAttr(resourceName, "spec.0.backend.*.virtual_service.0.client_policy.0.tls.0.ports.*", "8443"),
					resource.TestCheckTypeSetElemAttr(resourceName, "spec.0.backend.*.virtual_service.0.client_policy.0.tls.0.validation.0.subject_alternative_names.0.match.0.exact.*", "abc.example.com"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.connection_pool.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.health_check.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.outlier_detection.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.0.port", "8080"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.0.protocol", "http"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.logging.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.dns.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.dns.0.hostname", "serviceb.simpleapp.local"),
					resource.TestCheckResourceAttrSet(resourceName, "created_date"),
					resource.TestCheckResourceAttrSet(resourceName, "last_updated_date"),
					acctest.CheckResourceAttrAccountID(resourceName, "resource_owner"),
					acctest.CheckResourceAttrRegionalARN(resourceName, "arn", "appmesh", fmt.Sprintf("mesh/%s/virtualNode/%s", meshName, vnName)),
				),
			},
			{
				ResourceName:      resourceName,
				ImportStateId:     fmt.Sprintf("%s/%s", meshName, vnName),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccAwsAppmeshVirtualNode_backendDefaults(t *testing.T) {
	var vn appmesh.VirtualNodeData
	resourceName := "aws_appmesh_virtual_node.test"
	meshName := sdkacctest.RandomWithPrefix("tf-acc-test")
	vnName := sdkacctest.RandomWithPrefix("tf-acc-test")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); acctest.PreCheckPartitionHasService(appmesh.EndpointsID, t) },
		ErrorCheck:   acctest.ErrorCheck(t, appmesh.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAppmeshVirtualNodeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAppmeshVirtualNodeConfig_backendDefaults(meshName, vnName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAppmeshVirtualNodeExists(resourceName, &vn),
					resource.TestCheckResourceAttr(resourceName, "name", vnName),
					resource.TestCheckResourceAttr(resourceName, "mesh_name", meshName),
					acctest.CheckResourceAttrAccountID(resourceName, "mesh_owner"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.certificate.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.enforce", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.ports.#", "1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.ports.*", "8443"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.validation.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.validation.0.subject_alternative_names.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.validation.0.trust.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.validation.0.trust.0.acm.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.validation.0.trust.0.file.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.validation.0.trust.0.file.0.certificate_chain", "/cert_chain.pem"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.validation.0.trust.0.sds.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.logging.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.#", "0"),
					resource.TestCheckResourceAttrSet(resourceName, "created_date"),
					resource.TestCheckResourceAttrSet(resourceName, "last_updated_date"),
					acctest.CheckResourceAttrAccountID(resourceName, "resource_owner"),
					acctest.CheckResourceAttrRegionalARN(resourceName, "arn", "appmesh", fmt.Sprintf("mesh/%s/virtualNode/%s", meshName, vnName)),
				),
			},
			{
				Config: testAccAppmeshVirtualNodeConfig_backendDefaultsUpdated(meshName, vnName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAppmeshVirtualNodeExists(resourceName, &vn),
					resource.TestCheckResourceAttr(resourceName, "name", vnName),
					resource.TestCheckResourceAttr(resourceName, "mesh_name", meshName),
					acctest.CheckResourceAttrAccountID(resourceName, "mesh_owner"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.certificate.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.enforce", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.ports.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.ports.*", "443"),
					resource.TestCheckTypeSetElemAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.ports.*", "8443"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.validation.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.validation.0.subject_alternative_names.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.validation.0.trust.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.validation.0.trust.0.acm.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.validation.0.trust.0.file.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.validation.0.trust.0.file.0.certificate_chain", "/etc/ssl/certs/cert_chain.pem"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.validation.0.trust.0.sds.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.logging.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.#", "0"),
					resource.TestCheckResourceAttrSet(resourceName, "created_date"),
					resource.TestCheckResourceAttrSet(resourceName, "last_updated_date"),
					acctest.CheckResourceAttrAccountID(resourceName, "resource_owner"),
					acctest.CheckResourceAttrRegionalARN(resourceName, "arn", "appmesh", fmt.Sprintf("mesh/%s/virtualNode/%s", meshName, vnName)),
				),
			},
			{
				ResourceName:      resourceName,
				ImportStateId:     fmt.Sprintf("%s/%s", meshName, vnName),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccAwsAppmeshVirtualNode_backendDefaultsCertificate(t *testing.T) {
	var vn appmesh.VirtualNodeData
	resourceName := "aws_appmesh_virtual_node.test"
	meshName := sdkacctest.RandomWithPrefix("tf-acc-test")
	vnName := sdkacctest.RandomWithPrefix("tf-acc-test")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); acctest.PreCheckPartitionHasService(appmesh.EndpointsID, t) },
		ErrorCheck:   acctest.ErrorCheck(t, appmesh.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAppmeshVirtualNodeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAppmeshVirtualNodeConfig_backendDefaultsCertificate(meshName, vnName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAppmeshVirtualNodeExists(resourceName, &vn),
					resource.TestCheckResourceAttr(resourceName, "name", vnName),
					resource.TestCheckResourceAttr(resourceName, "mesh_name", meshName),
					acctest.CheckResourceAttrAccountID(resourceName, "mesh_owner"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.certificate.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.certificate.0.file.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.certificate.0.file.0.certificate_chain", "/cert_chain.pem"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.certificate.0.file.0.private_key", "tell-nobody"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.certificate.0.sds.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.enforce", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.ports.#", "1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.ports.*", "8443"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.validation.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.validation.0.subject_alternative_names.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.validation.0.subject_alternative_names.0.match.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.validation.0.subject_alternative_names.0.match.0.exact.#", "1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.validation.0.subject_alternative_names.0.match.0.exact.*", "def.example.com"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.validation.0.trust.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.validation.0.trust.0.acm.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.validation.0.trust.0.file.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.validation.0.trust.0.sds.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.0.client_policy.0.tls.0.validation.0.trust.0.sds.0.secret_name", "restricted"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.logging.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.#", "0"),
					resource.TestCheckResourceAttrSet(resourceName, "created_date"),
					resource.TestCheckResourceAttrSet(resourceName, "last_updated_date"),
					acctest.CheckResourceAttrAccountID(resourceName, "resource_owner"),
					acctest.CheckResourceAttrRegionalARN(resourceName, "arn", "appmesh", fmt.Sprintf("mesh/%s/virtualNode/%s", meshName, vnName)),
				),
			},
			{
				ResourceName:      resourceName,
				ImportStateId:     fmt.Sprintf("%s/%s", meshName, vnName),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccAwsAppmeshVirtualNode_cloudMapServiceDiscovery(t *testing.T) {
	var vn appmesh.VirtualNodeData
	resourceName := "aws_appmesh_virtual_node.test"
	nsResourceName := "aws_service_discovery_http_namespace.test"
	meshName := sdkacctest.RandomWithPrefix("tf-acc-test")
	vnName := sdkacctest.RandomWithPrefix("tf-acc-test")
	// Avoid 'config is invalid: last character of "name" must be a letter' for aws_service_discovery_http_namespace.
	rName := fmt.Sprintf("tf-acc-test-%s", sdkacctest.RandStringFromCharSet(20, sdkacctest.CharSetAlpha))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); acctest.PreCheckPartitionHasService(appmesh.EndpointsID, t) },
		ErrorCheck:   acctest.ErrorCheck(t, appmesh.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAppmeshVirtualNodeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAppmeshVirtualNodeConfig_cloudMapServiceDiscovery(meshName, vnName, rName, "Key1", "Value1"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAppmeshVirtualNodeExists(resourceName, &vn),
					resource.TestCheckResourceAttr(resourceName, "name", vnName),
					resource.TestCheckResourceAttr(resourceName, "mesh_name", meshName),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.aws_cloud_map.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.aws_cloud_map.0.attributes.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.aws_cloud_map.0.attributes.Key1", "Value1"),
					resource.TestCheckResourceAttrPair(resourceName, "spec.0.service_discovery.0.aws_cloud_map.0.namespace_name", nsResourceName, "name"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.aws_cloud_map.0.service_name", rName),
				),
			},
			{
				Config: testAccAppmeshVirtualNodeConfig_cloudMapServiceDiscovery(meshName, vnName, rName, "Key1", "Value2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAppmeshVirtualNodeExists(resourceName, &vn),
					resource.TestCheckResourceAttr(resourceName, "name", vnName),
					resource.TestCheckResourceAttr(resourceName, "mesh_name", meshName),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.aws_cloud_map.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.aws_cloud_map.0.attributes.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.aws_cloud_map.0.attributes.Key1", "Value2"),
					resource.TestCheckResourceAttrPair(resourceName, "spec.0.service_discovery.0.aws_cloud_map.0.namespace_name", nsResourceName, "name"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.aws_cloud_map.0.service_name", rName),
				),
			},
			{
				ResourceName:      resourceName,
				ImportStateId:     fmt.Sprintf("%s/%s", meshName, vnName),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccAwsAppmeshVirtualNode_listenerConnectionPool(t *testing.T) {
	var vn appmesh.VirtualNodeData
	resourceName := "aws_appmesh_virtual_node.test"
	meshName := sdkacctest.RandomWithPrefix("tf-acc-test")
	vnName := sdkacctest.RandomWithPrefix("tf-acc-test")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); acctest.PreCheckPartitionHasService(appmesh.EndpointsID, t) },
		ErrorCheck:   acctest.ErrorCheck(t, appmesh.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAppmeshVirtualNodeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAppmeshVirtualNodeConfig_listenerConnectionPool(meshName, vnName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAppmeshVirtualNodeExists(resourceName, &vn),
					resource.TestCheckResourceAttr(resourceName, "name", vnName),
					resource.TestCheckResourceAttr(resourceName, "mesh_name", meshName),
					acctest.CheckResourceAttrAccountID(resourceName, "mesh_owner"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "spec.0.backend.*", map[string]string{
						"virtual_service.#":                      "1",
						"virtual_service.0.virtual_service_name": "servicea.simpleapp.local",
					}),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.connection_pool.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.connection_pool.0.grpc.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.connection_pool.0.http.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.connection_pool.0.http2.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.connection_pool.0.tcp.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.connection_pool.0.tcp.0.max_connections", "4"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.0.port", "8080"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.0.protocol", "tcp"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.logging.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.dns.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.dns.0.hostname", "serviceb.simpleapp.local"),
					resource.TestCheckResourceAttrSet(resourceName, "created_date"),
					resource.TestCheckResourceAttrSet(resourceName, "last_updated_date"),
					acctest.CheckResourceAttrAccountID(resourceName, "resource_owner"),
					acctest.CheckResourceAttrRegionalARN(resourceName, "arn", "appmesh", fmt.Sprintf("mesh/%s/virtualNode/%s", meshName, vnName)),
				),
			},
			{
				Config: testAccAppmeshVirtualNodeConfig_listenerConnectionPoolUpdated(meshName, vnName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAppmeshVirtualNodeExists(resourceName, &vn),
					resource.TestCheckResourceAttr(resourceName, "name", vnName),
					resource.TestCheckResourceAttr(resourceName, "mesh_name", meshName),
					acctest.CheckResourceAttrAccountID(resourceName, "mesh_owner"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "spec.0.backend.*", map[string]string{
						"virtual_service.#":                      "1",
						"virtual_service.0.virtual_service_name": "servicea.simpleapp.local",
					}),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.connection_pool.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.connection_pool.0.grpc.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.connection_pool.0.http.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.connection_pool.0.http.0.max_connections", "8"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.connection_pool.0.http.0.max_pending_requests", "16"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.connection_pool.0.http2.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.connection_pool.0.tcp.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.0.port", "8080"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.0.protocol", "http"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.logging.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.dns.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.dns.0.hostname", "serviceb.simpleapp.local"),
					resource.TestCheckResourceAttrSet(resourceName, "created_date"),
					resource.TestCheckResourceAttrSet(resourceName, "last_updated_date"),
					acctest.CheckResourceAttrAccountID(resourceName, "resource_owner"),
					acctest.CheckResourceAttrRegionalARN(resourceName, "arn", "appmesh", fmt.Sprintf("mesh/%s/virtualNode/%s", meshName, vnName)),
				),
			},
			{
				ResourceName:      resourceName,
				ImportStateId:     fmt.Sprintf("%s/%s", meshName, vnName),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccAwsAppmeshVirtualNode_listenerHealthChecks(t *testing.T) {
	var vn appmesh.VirtualNodeData
	resourceName := "aws_appmesh_virtual_node.test"
	meshName := sdkacctest.RandomWithPrefix("tf-acc-test")
	vnName := sdkacctest.RandomWithPrefix("tf-acc-test")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); acctest.PreCheckPartitionHasService(appmesh.EndpointsID, t) },
		ErrorCheck:   acctest.ErrorCheck(t, appmesh.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAppmeshVirtualNodeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAppmeshVirtualNodeConfig_listenerHealthChecks(meshName, vnName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAppmeshVirtualNodeExists(resourceName, &vn),
					resource.TestCheckResourceAttr(resourceName, "name", vnName),
					resource.TestCheckResourceAttr(resourceName, "mesh_name", meshName),
					acctest.CheckResourceAttrAccountID(resourceName, "mesh_owner"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "spec.0.backend.*", map[string]string{
						"virtual_service.#":                      "1",
						"virtual_service.0.client_policy.#":      "0",
						"virtual_service.0.virtual_service_name": "servicea.simpleapp.local",
					}),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.connection_pool.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.health_check.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.health_check.0.healthy_threshold", "3"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.health_check.0.interval_millis", "5000"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.health_check.0.path", "/ping"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.health_check.0.port", "8080"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.health_check.0.protocol", "http2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.health_check.0.timeout_millis", "2000"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.health_check.0.unhealthy_threshold", "5"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.outlier_detection.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.0.port", "8080"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.0.protocol", "grpc"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.timeout.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.logging.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.dns.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.dns.0.hostname", "serviceb.simpleapp.local"),
					resource.TestCheckResourceAttrSet(resourceName, "created_date"),
					resource.TestCheckResourceAttrSet(resourceName, "last_updated_date"),
					acctest.CheckResourceAttrAccountID(resourceName, "resource_owner"),
					acctest.CheckResourceAttrRegionalARN(resourceName, "arn", "appmesh", fmt.Sprintf("mesh/%s/virtualNode/%s", meshName, vnName)),
				),
			},
			{
				Config: testAccAppmeshVirtualNodeConfig_listenerHealthChecksUpdated(meshName, vnName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAppmeshVirtualNodeExists(resourceName, &vn),
					resource.TestCheckResourceAttr(resourceName, "name", vnName),
					resource.TestCheckResourceAttr(resourceName, "mesh_name", meshName),
					acctest.CheckResourceAttrAccountID(resourceName, "mesh_owner"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "spec.0.backend.*", map[string]string{
						"virtual_service.#":                      "1",
						"virtual_service.0.client_policy.#":      "0",
						"virtual_service.0.virtual_service_name": "servicec.simpleapp.local",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "spec.0.backend.*", map[string]string{
						"virtual_service.#":                      "1",
						"virtual_service.0.client_policy.#":      "0",
						"virtual_service.0.virtual_service_name": "serviced.simpleapp.local",
					}),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.connection_pool.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.health_check.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.health_check.0.healthy_threshold", "4"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.health_check.0.interval_millis", "7000"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.health_check.0.port", "8081"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.health_check.0.protocol", "tcp"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.health_check.0.timeout_millis", "3000"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.health_check.0.unhealthy_threshold", "9"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.outlier_detection.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.0.port", "8081"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.0.protocol", "http"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.timeout.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.logging.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.dns.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.dns.0.hostname", "serviceb1.simpleapp.local"),
					resource.TestCheckResourceAttrSet(resourceName, "created_date"),
					resource.TestCheckResourceAttrSet(resourceName, "last_updated_date"),
					acctest.CheckResourceAttrAccountID(resourceName, "resource_owner"),
					acctest.CheckResourceAttrRegionalARN(resourceName, "arn", "appmesh", fmt.Sprintf("mesh/%s/virtualNode/%s", meshName, vnName)),
				),
			},
			{
				ResourceName:      resourceName,
				ImportStateId:     fmt.Sprintf("%s/%s", meshName, vnName),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccAwsAppmeshVirtualNode_listenerOutlierDetection(t *testing.T) {
	var vn appmesh.VirtualNodeData
	resourceName := "aws_appmesh_virtual_node.test"
	meshName := sdkacctest.RandomWithPrefix("tf-acc-test")
	vnName := sdkacctest.RandomWithPrefix("tf-acc-test")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); acctest.PreCheckPartitionHasService(appmesh.EndpointsID, t) },
		ErrorCheck:   acctest.ErrorCheck(t, appmesh.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAppmeshVirtualNodeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAppmeshVirtualNodeConfig_listenerOutlierDetection(meshName, vnName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAppmeshVirtualNodeExists(resourceName, &vn),
					resource.TestCheckResourceAttr(resourceName, "name", vnName),
					resource.TestCheckResourceAttr(resourceName, "mesh_name", meshName),
					acctest.CheckResourceAttrAccountID(resourceName, "mesh_owner"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "spec.0.backend.*", map[string]string{
						"virtual_service.#":                      "1",
						"virtual_service.0.virtual_service_name": "servicea.simpleapp.local",
					}),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.outlier_detection.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.outlier_detection.0.base_ejection_duration.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.outlier_detection.0.base_ejection_duration.0.unit", "ms"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.outlier_detection.0.base_ejection_duration.0.value", "250000"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.outlier_detection.0.interval.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.outlier_detection.0.interval.0.unit", "s"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.outlier_detection.0.interval.0.value", "10"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.outlier_detection.0.max_ejection_percent", "50"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.outlier_detection.0.max_server_errors", "5"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.0.port", "8080"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.0.protocol", "tcp"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.logging.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.dns.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.dns.0.hostname", "serviceb.simpleapp.local"),
					resource.TestCheckResourceAttrSet(resourceName, "created_date"),
					resource.TestCheckResourceAttrSet(resourceName, "last_updated_date"),
					acctest.CheckResourceAttrAccountID(resourceName, "resource_owner"),
					acctest.CheckResourceAttrRegionalARN(resourceName, "arn", "appmesh", fmt.Sprintf("mesh/%s/virtualNode/%s", meshName, vnName)),
				),
			},
			{
				Config: testAccAppmeshVirtualNodeConfig_listenerOutlierDetectionUpdated(meshName, vnName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAppmeshVirtualNodeExists(resourceName, &vn),
					resource.TestCheckResourceAttr(resourceName, "name", vnName),
					resource.TestCheckResourceAttr(resourceName, "mesh_name", meshName),
					acctest.CheckResourceAttrAccountID(resourceName, "mesh_owner"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "spec.0.backend.*", map[string]string{
						"virtual_service.#":                      "1",
						"virtual_service.0.virtual_service_name": "servicea.simpleapp.local",
					}),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.outlier_detection.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.outlier_detection.0.base_ejection_duration.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.outlier_detection.0.base_ejection_duration.0.unit", "s"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.outlier_detection.0.base_ejection_duration.0.value", "6"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.outlier_detection.0.interval.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.outlier_detection.0.interval.0.unit", "ms"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.outlier_detection.0.interval.0.value", "10000"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.outlier_detection.0.max_ejection_percent", "60"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.outlier_detection.0.max_server_errors", "6"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.0.port", "8080"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.0.protocol", "http"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.logging.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.dns.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.dns.0.hostname", "serviceb.simpleapp.local"),
					resource.TestCheckResourceAttrSet(resourceName, "created_date"),
					resource.TestCheckResourceAttrSet(resourceName, "last_updated_date"),
					acctest.CheckResourceAttrAccountID(resourceName, "resource_owner"),
					acctest.CheckResourceAttrRegionalARN(resourceName, "arn", "appmesh", fmt.Sprintf("mesh/%s/virtualNode/%s", meshName, vnName)),
				),
			},
			{
				ResourceName:      resourceName,
				ImportStateId:     fmt.Sprintf("%s/%s", meshName, vnName),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccAwsAppmeshVirtualNode_listenerTimeout(t *testing.T) {
	var vn appmesh.VirtualNodeData
	resourceName := "aws_appmesh_virtual_node.test"
	meshName := sdkacctest.RandomWithPrefix("tf-acc-test")
	vnName := sdkacctest.RandomWithPrefix("tf-acc-test")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); acctest.PreCheckPartitionHasService(appmesh.EndpointsID, t) },
		ErrorCheck:   acctest.ErrorCheck(t, appmesh.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAppmeshVirtualNodeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAppmeshVirtualNodeConfig_listenerTimeout(meshName, vnName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAppmeshVirtualNodeExists(resourceName, &vn),
					resource.TestCheckResourceAttr(resourceName, "name", vnName),
					resource.TestCheckResourceAttr(resourceName, "mesh_name", meshName),
					acctest.CheckResourceAttrAccountID(resourceName, "mesh_owner"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "spec.0.backend.*", map[string]string{
						"virtual_service.#":                      "1",
						"virtual_service.0.virtual_service_name": "servicea.simpleapp.local",
					}),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.0.port", "8080"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.0.protocol", "tcp"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.timeout.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.timeout.0.grpc.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.timeout.0.http.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.timeout.0.http2.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.timeout.0.tcp.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.timeout.0.tcp.0.idle.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.timeout.0.tcp.0.idle.0.unit", "ms"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.timeout.0.tcp.0.idle.0.value", "250000"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.logging.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.dns.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.dns.0.hostname", "serviceb.simpleapp.local"),
					resource.TestCheckResourceAttrSet(resourceName, "created_date"),
					resource.TestCheckResourceAttrSet(resourceName, "last_updated_date"),
					acctest.CheckResourceAttrAccountID(resourceName, "resource_owner"),
					acctest.CheckResourceAttrRegionalARN(resourceName, "arn", "appmesh", fmt.Sprintf("mesh/%s/virtualNode/%s", meshName, vnName)),
				),
			},
			{
				Config: testAccAppmeshVirtualNodeConfig_listenerTimeoutUpdated(meshName, vnName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAppmeshVirtualNodeExists(resourceName, &vn),
					resource.TestCheckResourceAttr(resourceName, "name", vnName),
					resource.TestCheckResourceAttr(resourceName, "mesh_name", meshName),
					acctest.CheckResourceAttrAccountID(resourceName, "mesh_owner"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "spec.0.backend.*", map[string]string{
						"virtual_service.#":                      "1",
						"virtual_service.0.virtual_service_name": "servicea.simpleapp.local",
					}),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.0.port", "8080"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.0.protocol", "http"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.timeout.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.timeout.0.grpc.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.timeout.0.http.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.timeout.0.http.0.idle.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.timeout.0.http.0.idle.0.unit", "s"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.timeout.0.http.0.idle.0.value", "10"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.timeout.0.http.0.per_request.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.timeout.0.http.0.per_request.0.unit", "s"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.timeout.0.http.0.per_request.0.value", "5"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.timeout.0.http2.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.timeout.0.tcp.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.logging.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.dns.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.dns.0.hostname", "serviceb.simpleapp.local"),
					resource.TestCheckResourceAttrSet(resourceName, "created_date"),
					resource.TestCheckResourceAttrSet(resourceName, "last_updated_date"),
					acctest.CheckResourceAttrAccountID(resourceName, "resource_owner"),
					acctest.CheckResourceAttrRegionalARN(resourceName, "arn", "appmesh", fmt.Sprintf("mesh/%s/virtualNode/%s", meshName, vnName)),
				),
			},
			{
				ResourceName:      resourceName,
				ImportStateId:     fmt.Sprintf("%s/%s", meshName, vnName),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccAwsAppmeshVirtualNode_listenerTls(t *testing.T) {
	var vn appmesh.VirtualNodeData
	var ca acmpca.CertificateAuthority
	resourceName := "aws_appmesh_virtual_node.test"
	acmCAResourceName := "aws_acmpca_certificate_authority.test"
	acmCertificateResourceName := "aws_acm_certificate.test"

	meshName := sdkacctest.RandomWithPrefix("tf-acc-test")
	vnName := sdkacctest.RandomWithPrefix("tf-acc-test")
	domain := acctest.RandomDomainName()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); acctest.PreCheckPartitionHasService(appmesh.EndpointsID, t) },
		ErrorCheck:   acctest.ErrorCheck(t, appmesh.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAppmeshVirtualNodeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAppmeshVirtualNodeConfig_listenerTlsFile(meshName, vnName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAppmeshVirtualNodeExists(resourceName, &vn),
					resource.TestCheckResourceAttr(resourceName, "name", vnName),
					resource.TestCheckResourceAttr(resourceName, "mesh_name", meshName),
					acctest.CheckResourceAttrAccountID(resourceName, "mesh_owner"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "spec.0.backend.*", map[string]string{
						"virtual_service.#":                      "1",
						"virtual_service.0.client_policy.#":      "0",
						"virtual_service.0.virtual_service_name": "servicea.simpleapp.local",
					}),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.connection_pool.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.health_check.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.outlier_detection.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.0.port", "8080"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.0.protocol", "http"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.certificate.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.certificate.0.acm.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.certificate.0.file.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.certificate.0.file.0.certificate_chain", "/cert_chain.pem"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.certificate.0.file.0.private_key", "/key.pem"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.certificate.0.sds.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.mode", "PERMISSIVE"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.validation.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.logging.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.dns.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.dns.0.hostname", "serviceb.simpleapp.local"),
					resource.TestCheckResourceAttrSet(resourceName, "created_date"),
					resource.TestCheckResourceAttrSet(resourceName, "last_updated_date"),
					acctest.CheckResourceAttrAccountID(resourceName, "resource_owner"),
					acctest.CheckResourceAttrRegionalARN(resourceName, "arn", "appmesh", fmt.Sprintf("mesh/%s/virtualNode/%s", meshName, vnName)),
				),
			},
			{
				ResourceName:      resourceName,
				ImportStateId:     fmt.Sprintf("%s/%s", meshName, vnName),
				ImportState:       true,
				ImportStateVerify: true,
			},
			// We need to create and activate the CA before issuing a certificate.
			{
				Config: testAccAppmeshVirtualNodeConfigRootCA(domain),
				Check: resource.ComposeTestCheckFunc(
					acctest.CheckACMPCACertificateAuthorityExists(acmCAResourceName, &ca),
					acctest.CheckACMPCACertificateAuthorityActivateCA(&ca),
				),
			},
			{
				Config: testAccAppmeshVirtualNodeConfig_listenerTlsAcm(meshName, vnName, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAppmeshVirtualNodeExists(resourceName, &vn),
					resource.TestCheckResourceAttr(resourceName, "name", vnName),
					resource.TestCheckResourceAttr(resourceName, "mesh_name", meshName),
					acctest.CheckResourceAttrAccountID(resourceName, "mesh_owner"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "spec.0.backend.*", map[string]string{
						"virtual_service.#":                      "1",
						"virtual_service.0.client_policy.#":      "0",
						"virtual_service.0.virtual_service_name": "servicea.simpleapp.local",
					}),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.connection_pool.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.health_check.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.outlier_detection.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.0.port", "8080"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.0.protocol", "http"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.certificate.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.certificate.0.acm.#", "1"),
					resource.TestCheckResourceAttrPair(resourceName, "spec.0.listener.0.tls.0.certificate.0.acm.0.certificate_arn", acmCertificateResourceName, "arn"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.certificate.0.file.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.certificate.0.sds.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.mode", "STRICT"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.validation.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.logging.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.dns.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.dns.0.hostname", "serviceb.simpleapp.local"),
					resource.TestCheckResourceAttrSet(resourceName, "created_date"),
					resource.TestCheckResourceAttrSet(resourceName, "last_updated_date"),
					acctest.CheckResourceAttrAccountID(resourceName, "resource_owner"),
					acctest.CheckResourceAttrRegionalARN(resourceName, "arn", "appmesh", fmt.Sprintf("mesh/%s/virtualNode/%s", meshName, vnName)),
				),
			},
			{
				ResourceName:      resourceName,
				ImportStateId:     fmt.Sprintf("%s/%s", meshName, vnName),
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAppmeshVirtualNodeConfig_listenerTlsAcm(meshName, vnName, domain),
				Check: resource.ComposeTestCheckFunc(
					// CA must be DISABLED for deletion.
					acctest.CheckACMPCACertificateAuthorityDisableCA(&ca),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccAwsAppmeshVirtualNode_listenerValidation(t *testing.T) {
	var vn appmesh.VirtualNodeData
	resourceName := "aws_appmesh_virtual_node.test"
	meshName := sdkacctest.RandomWithPrefix("tf-acc-test")
	vnName := sdkacctest.RandomWithPrefix("tf-acc-test")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); acctest.PreCheckPartitionHasService(appmesh.EndpointsID, t) },
		ErrorCheck:   acctest.ErrorCheck(t, appmesh.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAppmeshVirtualNodeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAppmeshVirtualNodeConfig_listenerValidation(meshName, vnName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAppmeshVirtualNodeExists(resourceName, &vn),
					resource.TestCheckResourceAttr(resourceName, "name", vnName),
					resource.TestCheckResourceAttr(resourceName, "mesh_name", meshName),
					acctest.CheckResourceAttrAccountID(resourceName, "mesh_owner"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "spec.0.backend.*", map[string]string{
						"virtual_service.#":                      "1",
						"virtual_service.0.client_policy.#":      "0",
						"virtual_service.0.virtual_service_name": "servicea.simpleapp.local",
					}),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.connection_pool.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.health_check.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.outlier_detection.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.0.port", "8080"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.0.protocol", "http"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.certificate.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.certificate.0.acm.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.certificate.0.file.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.certificate.0.sds.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.certificate.0.sds.0.secret_name", "very-secret"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.mode", "PERMISSIVE"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.validation.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.validation.0.subject_alternative_names.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.validation.0.subject_alternative_names.0.match.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.validation.0.subject_alternative_names.0.match.0.exact.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "spec.0.listener.0.tls.0.validation.0.subject_alternative_names.0.match.0.exact.*", "abc.example.com"),
					resource.TestCheckTypeSetElemAttr(resourceName, "spec.0.listener.0.tls.0.validation.0.subject_alternative_names.0.match.0.exact.*", "xyz.example.com"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.validation.0.trust.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.validation.0.trust.0.acm.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.validation.0.trust.0.file.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.validation.0.trust.0.file.0.certificate_chain", "/cert_chain.pem"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.validation.0.trust.0.sds.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.logging.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.dns.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.dns.0.hostname", "serviceb.simpleapp.local"),
					resource.TestCheckResourceAttrSet(resourceName, "created_date"),
					resource.TestCheckResourceAttrSet(resourceName, "last_updated_date"),
					acctest.CheckResourceAttrAccountID(resourceName, "resource_owner"),
					acctest.CheckResourceAttrRegionalARN(resourceName, "arn", "appmesh", fmt.Sprintf("mesh/%s/virtualNode/%s", meshName, vnName)),
				),
			},
			{
				ResourceName:      resourceName,
				ImportStateId:     fmt.Sprintf("%s/%s", meshName, vnName),
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAppmeshVirtualNodeConfig_listenerValidationUpdated(meshName, vnName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAppmeshVirtualNodeExists(resourceName, &vn),
					resource.TestCheckResourceAttr(resourceName, "name", vnName),
					resource.TestCheckResourceAttr(resourceName, "mesh_name", meshName),
					acctest.CheckResourceAttrAccountID(resourceName, "mesh_owner"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "spec.0.backend.*", map[string]string{
						"virtual_service.#":                      "1",
						"virtual_service.0.client_policy.#":      "0",
						"virtual_service.0.virtual_service_name": "servicea.simpleapp.local",
					}),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend_defaults.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.connection_pool.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.health_check.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.outlier_detection.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.0.port", "8080"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.port_mapping.0.protocol", "http"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.certificate.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.certificate.0.acm.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.certificate.0.file.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.certificate.0.sds.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.certificate.0.sds.0.secret_name", "top-secret"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.mode", "PERMISSIVE"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.validation.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.validation.0.subject_alternative_names.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.validation.0.trust.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.validation.0.trust.0.acm.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.validation.0.trust.0.file.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.validation.0.trust.0.sds.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listener.0.tls.0.validation.0.trust.0.sds.0.secret_name", "confidential"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.logging.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.dns.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_discovery.0.dns.0.hostname", "serviceb.simpleapp.local"),
					resource.TestCheckResourceAttrSet(resourceName, "created_date"),
					resource.TestCheckResourceAttrSet(resourceName, "last_updated_date"),
					acctest.CheckResourceAttrAccountID(resourceName, "resource_owner"),
					acctest.CheckResourceAttrRegionalARN(resourceName, "arn", "appmesh", fmt.Sprintf("mesh/%s/virtualNode/%s", meshName, vnName)),
				),
			},
		},
	})
}

func testAccAwsAppmeshVirtualNode_logging(t *testing.T) {
	var vn appmesh.VirtualNodeData
	resourceName := "aws_appmesh_virtual_node.test"
	meshName := sdkacctest.RandomWithPrefix("tf-acc-test")
	vnName := sdkacctest.RandomWithPrefix("tf-acc-test")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); acctest.PreCheckPartitionHasService(appmesh.EndpointsID, t) },
		ErrorCheck:   acctest.ErrorCheck(t, appmesh.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAppmeshVirtualNodeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAppmeshVirtualNodeConfig_logging(meshName, vnName, "/dev/stdout"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAppmeshVirtualNodeExists(resourceName, &vn),
					resource.TestCheckResourceAttr(resourceName, "name", vnName),
					resource.TestCheckResourceAttr(resourceName, "mesh_name", meshName),
					acctest.CheckResourceAttrAccountID(resourceName, "mesh_owner"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.logging.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.logging.0.access_log.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.logging.0.access_log.0.file.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.logging.0.access_log.0.file.0.path", "/dev/stdout"),
				),
			},
			{
				Config: testAccAppmeshVirtualNodeConfig_logging(meshName, vnName, "/tmp/access.log"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAppmeshVirtualNodeExists(resourceName, &vn),
					resource.TestCheckResourceAttr(resourceName, "name", vnName),
					resource.TestCheckResourceAttr(resourceName, "mesh_name", meshName),
					acctest.CheckResourceAttrAccountID(resourceName, "mesh_owner"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.logging.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.logging.0.access_log.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.logging.0.access_log.0.file.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.logging.0.access_log.0.file.0.path", "/tmp/access.log"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportStateId:     fmt.Sprintf("%s/%s", meshName, vnName),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccAwsAppmeshVirtualNode_tags(t *testing.T) {
	var vn appmesh.VirtualNodeData
	resourceName := "aws_appmesh_virtual_node.test"
	meshName := sdkacctest.RandomWithPrefix("tf-acc-test")
	vnName := sdkacctest.RandomWithPrefix("tf-acc-test")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); acctest.PreCheckPartitionHasService(appmesh.EndpointsID, t) },
		ErrorCheck:   acctest.ErrorCheck(t, appmesh.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAppmeshVirtualNodeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAppmeshVirtualNodeConfig_tags(meshName, vnName, "foo", "bar", "good", "bad"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAppmeshVirtualNodeExists(resourceName, &vn),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.foo", "bar"),
					resource.TestCheckResourceAttr(resourceName, "tags.good", "bad"),
				),
			},
			{
				Config: testAccAppmeshVirtualNodeConfig_tags(meshName, vnName, "foo2", "bar", "good", "bad2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAppmeshVirtualNodeExists(resourceName, &vn),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.foo2", "bar"),
					resource.TestCheckResourceAttr(resourceName, "tags.good", "bad2"),
				),
			},
			{
				Config: testAccAppmeshVirtualNodeConfig_basic(meshName, vnName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAppmeshVirtualNodeExists(resourceName, &vn),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "0"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportStateId:     fmt.Sprintf("%s/%s", meshName, vnName),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckAppmeshVirtualNodeDestroy(s *terraform.State) error {
	conn := acctest.Provider.Meta().(*conns.AWSClient).AppMeshConn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_appmesh_virtual_node" {
			continue
		}

		_, err := conn.DescribeVirtualNode(&appmesh.DescribeVirtualNodeInput{
			MeshName:        aws.String(rs.Primary.Attributes["mesh_name"]),
			VirtualNodeName: aws.String(rs.Primary.Attributes["name"]),
		})
		if tfawserr.ErrMessageContains(err, appmesh.ErrCodeNotFoundException, "") {
			continue
		}
		if err != nil {
			return err
		}
		return fmt.Errorf("still exist.")
	}

	return nil
}

func testAccCheckAppmeshVirtualNodeExists(name string, v *appmesh.VirtualNodeData) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := acctest.Provider.Meta().(*conns.AWSClient).AppMeshConn

		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		resp, err := conn.DescribeVirtualNode(&appmesh.DescribeVirtualNodeInput{
			MeshName:        aws.String(rs.Primary.Attributes["mesh_name"]),
			VirtualNodeName: aws.String(rs.Primary.Attributes["name"]),
		})
		if err != nil {
			return err
		}

		*v = *resp.VirtualNode

		return nil
	}
}

func testAccAppmeshVirtualNodeConfig_mesh(rName string) string {
	return fmt.Sprintf(`
resource "aws_appmesh_mesh" "test" {
  name = %[1]q
}
`, rName)
}

func testAccAppmeshVirtualNodeConfigRootCA(domain string) string {
	return fmt.Sprintf(`
resource "aws_acmpca_certificate_authority" "test" {
  permanent_deletion_time_in_days = 7
  type                            = "ROOT"

  certificate_authority_configuration {
    key_algorithm     = "RSA_4096"
    signing_algorithm = "SHA512WITHRSA"

    subject {
      common_name = %[1]q
    }
  }
}
`, domain)
}

func testAccAppmeshVirtualNodeConfigPrivateCert(domain string) string {
	return fmt.Sprintf(`
resource "aws_acm_certificate" "test" {
  domain_name               = "test.%[1]s"
  certificate_authority_arn = aws_acmpca_certificate_authority.test.arn
}
`, domain)
}

func testAccAppmeshVirtualNodeConfig_basic(meshName, vnName string) string {
	return acctest.ConfigCompose(testAccAppmeshVirtualNodeConfig_mesh(meshName), fmt.Sprintf(`
resource "aws_appmesh_virtual_node" "test" {
  name      = %[1]q
  mesh_name = aws_appmesh_mesh.test.id

  spec {}
}
`, vnName))
}

func testAccAppmeshVirtualNodeConfig_backendDefaults(meshName, vnName string) string {
	return acctest.ConfigCompose(testAccAppmeshVirtualNodeConfig_mesh(meshName), fmt.Sprintf(`
resource "aws_appmesh_virtual_node" "test" {
  name      = %[1]q
  mesh_name = aws_appmesh_mesh.test.id

  spec {
    backend_defaults {
      client_policy {
        tls {
          ports = [8443]

          validation {
            trust {
              file {
                certificate_chain = "/cert_chain.pem"
              }
            }
          }
        }
      }
    }
  }
}
`, vnName))
}

func testAccAppmeshVirtualNodeConfig_backendDefaultsUpdated(meshName, vnName string) string {
	return acctest.ConfigCompose(testAccAppmeshVirtualNodeConfig_mesh(meshName), fmt.Sprintf(`
resource "aws_appmesh_virtual_node" "test" {
  name      = %[1]q
  mesh_name = aws_appmesh_mesh.test.id

  spec {
    backend_defaults {
      client_policy {
        tls {
          ports = [443, 8443]

          validation {
            trust {
              file {
                certificate_chain = "/etc/ssl/certs/cert_chain.pem"
              }
            }
          }
        }
      }
    }
  }
}
`, vnName))
}

func testAccAppmeshVirtualNodeConfig_backendDefaultsCertificate(meshName, vnName string) string {
	return acctest.ConfigCompose(testAccAppmeshVirtualNodeConfig_mesh(meshName), fmt.Sprintf(`
resource "aws_appmesh_virtual_node" "test" {
  name      = %[1]q
  mesh_name = aws_appmesh_mesh.test.id

  spec {
    backend_defaults {
      client_policy {
        tls {
          certificate {
            file {
              certificate_chain = "/cert_chain.pem"
              private_key       = "tell-nobody"
            }
          }

          ports = [8443]

          validation {
            subject_alternative_names {
              match {
                exact = ["def.example.com"]
              }
            }

            trust {
              sds {
                secret_name = "restricted"
              }
            }
          }
        }
      }
    }
  }
}
`, vnName))
}

func testAccAppmeshVirtualNodeConfig_backendClientPolicyAcm(meshName, vnName, domain string) string {
	return acctest.ConfigCompose(
		testAccAppmeshVirtualNodeConfigRootCA(domain),
		testAccAppmeshVirtualNodeConfigPrivateCert(domain),
		testAccAppmeshVirtualNodeConfig_mesh(meshName),
		fmt.Sprintf(`
resource "aws_appmesh_virtual_node" "test" {
  name      = %[1]q
  mesh_name = aws_appmesh_mesh.test.id

  spec {
    backend {
      virtual_service {
        virtual_service_name = "servicea.simpleapp.local"

        client_policy {
          tls {
            ports = [8443]

            validation {
              trust {
                acm {
                  certificate_authority_arns = [aws_acmpca_certificate_authority.test.arn]
                }
              }
            }
          }
        }
      }
    }

    listener {
      port_mapping {
        port     = 8080
        protocol = "http"
      }
    }

    service_discovery {
      dns {
        hostname = "serviceb.simpleapp.local"
      }
    }
  }
}
`, vnName))
}

func testAccAppmeshVirtualNodeConfig_backendClientPolicyFile(meshName, vnName string) string {
	return acctest.ConfigCompose(testAccAppmeshVirtualNodeConfig_mesh(meshName), fmt.Sprintf(`
resource "aws_appmesh_virtual_node" "test" {
  name      = %[1]q
  mesh_name = aws_appmesh_mesh.test.id

  spec {
    backend {
      virtual_service {
        virtual_service_name = "servicea.simpleapp.local"

        client_policy {
          tls {
            ports = [8443]

            validation {
              trust {
                file {
                  certificate_chain = "/cert_chain.pem"
                }
              }
            }
          }
        }
      }
    }

    listener {
      port_mapping {
        port     = 8080
        protocol = "http"
      }
    }

    service_discovery {
      dns {
        hostname = "serviceb.simpleapp.local"
      }
    }
  }
}
`, vnName))
}

func testAccAppmeshVirtualNodeConfig_backendClientPolicyFileUpdated(meshName, vnName string) string {
	return acctest.ConfigCompose(testAccAppmeshVirtualNodeConfig_mesh(meshName), fmt.Sprintf(`
resource "aws_appmesh_virtual_node" "test" {
  name      = %[1]q
  mesh_name = aws_appmesh_mesh.test.id

  spec {
    backend {
      virtual_service {
        virtual_service_name = "servicea.simpleapp.local"

        client_policy {
          tls {
            ports = [443, 8443]

            validation {
              subject_alternative_names {
                match {
                  exact = ["abc.example.com"]
                }
              }

              trust {
                file {
                  certificate_chain = "/etc/ssl/certs/cert_chain.pem"
                }
              }
            }
          }
        }
      }
    }

    listener {
      port_mapping {
        port     = 8080
        protocol = "http"
      }
    }

    service_discovery {
      dns {
        hostname = "serviceb.simpleapp.local"
      }
    }
  }
}
`, vnName))
}

func testAccAppmeshVirtualNodeConfig_cloudMapServiceDiscovery(meshName, vnName, rName, attrKey, attrValue string) string {
	return acctest.ConfigCompose(testAccAppmeshVirtualNodeConfig_mesh(meshName), fmt.Sprintf(`
resource "aws_service_discovery_http_namespace" "test" {
  name = %[2]q
}

resource "aws_appmesh_virtual_node" "test" {
  name      = %[1]q
  mesh_name = aws_appmesh_mesh.test.id

  spec {
    backend {
      virtual_service {
        virtual_service_name = "servicea.simpleapp.local"
      }
    }

    listener {
      port_mapping {
        port     = 8080
        protocol = "http"
      }
    }

    service_discovery {
      aws_cloud_map {
        attributes = {
          %[3]s = %[4]q
        }

        service_name   = %[2]q
        namespace_name = aws_service_discovery_http_namespace.test.name
      }
    }
  }
}
`, vnName, rName, attrKey, attrValue))
}

func testAccAppmeshVirtualNodeConfig_listenerConnectionPool(meshName, vnName string) string {
	return acctest.ConfigCompose(testAccAppmeshVirtualNodeConfig_mesh(meshName), fmt.Sprintf(`
resource "aws_appmesh_virtual_node" "test" {
  name      = %[1]q
  mesh_name = aws_appmesh_mesh.test.id

  spec {
    backend {
      virtual_service {
        virtual_service_name = "servicea.simpleapp.local"
      }
    }

    listener {
      port_mapping {
        port     = 8080
        protocol = "tcp"
      }

      connection_pool {
        tcp {
          max_connections = 4
        }
      }
    }

    service_discovery {
      dns {
        hostname = "serviceb.simpleapp.local"
      }
    }
  }
}
`, vnName))
}

func testAccAppmeshVirtualNodeConfig_listenerConnectionPoolUpdated(meshName, vnName string) string {
	return acctest.ConfigCompose(testAccAppmeshVirtualNodeConfig_mesh(meshName), fmt.Sprintf(`
resource "aws_appmesh_virtual_node" "test" {
  name      = %[1]q
  mesh_name = aws_appmesh_mesh.test.id

  spec {
    backend {
      virtual_service {
        virtual_service_name = "servicea.simpleapp.local"
      }
    }

    listener {
      port_mapping {
        port     = 8080
        protocol = "http"
      }

      connection_pool {
        http {
          max_connections      = 8
          max_pending_requests = 16
        }
      }
    }

    service_discovery {
      dns {
        hostname = "serviceb.simpleapp.local"
      }
    }
  }
}
`, vnName))
}

func testAccAppmeshVirtualNodeConfig_listenerHealthChecks(meshName, vnName string) string {
	return acctest.ConfigCompose(testAccAppmeshVirtualNodeConfig_mesh(meshName), fmt.Sprintf(`
resource "aws_appmesh_virtual_node" "test" {
  name      = %[1]q
  mesh_name = aws_appmesh_mesh.test.id

  spec {
    backend {
      virtual_service {
        virtual_service_name = "servicea.simpleapp.local"
      }
    }

    listener {
      port_mapping {
        port     = 8080
        protocol = "grpc"
      }

      health_check {
        protocol            = "http2"
        path                = "/ping"
        healthy_threshold   = 3
        unhealthy_threshold = 5
        timeout_millis      = 2000
        interval_millis     = 5000
      }
    }

    service_discovery {
      dns {
        hostname = "serviceb.simpleapp.local"
      }
    }
  }
}
`, vnName))
}

func testAccAppmeshVirtualNodeConfig_listenerHealthChecksUpdated(meshName, vnName string) string {
	return acctest.ConfigCompose(testAccAppmeshVirtualNodeConfig_mesh(meshName), fmt.Sprintf(`
resource "aws_appmesh_virtual_node" "test" {
  name      = %[1]q
  mesh_name = aws_appmesh_mesh.test.id

  spec {
    backend {
      virtual_service {
        virtual_service_name = "servicec.simpleapp.local"
      }
    }

    backend {
      virtual_service {
        virtual_service_name = "serviced.simpleapp.local"
      }
    }

    listener {
      port_mapping {
        port     = 8081
        protocol = "http"
      }

      health_check {
        protocol            = "tcp"
        port                = 8081
        healthy_threshold   = 4
        unhealthy_threshold = 9
        timeout_millis      = 3000
        interval_millis     = 7000
      }
    }

    service_discovery {
      dns {
        hostname = "serviceb1.simpleapp.local"
      }
    }
  }
}
`, vnName))
}

func testAccAppmeshVirtualNodeConfig_listenerOutlierDetection(meshName, vnName string) string {
	return acctest.ConfigCompose(testAccAppmeshVirtualNodeConfig_mesh(meshName), fmt.Sprintf(`
resource "aws_appmesh_virtual_node" "test" {
  name      = %[1]q
  mesh_name = aws_appmesh_mesh.test.id

  spec {
    backend {
      virtual_service {
        virtual_service_name = "servicea.simpleapp.local"
      }
    }

    listener {
      port_mapping {
        port     = 8080
        protocol = "tcp"
      }

      outlier_detection {
        base_ejection_duration {
          unit  = "ms"
          value = 250000
        }

        interval {
          unit  = "s"
          value = 10
        }

        max_ejection_percent = 50
        max_server_errors    = 5
      }
    }

    service_discovery {
      dns {
        hostname = "serviceb.simpleapp.local"
      }
    }
  }
}
`, vnName))
}

func testAccAppmeshVirtualNodeConfig_listenerOutlierDetectionUpdated(meshName, vnName string) string {
	return acctest.ConfigCompose(testAccAppmeshVirtualNodeConfig_mesh(meshName), fmt.Sprintf(`
resource "aws_appmesh_virtual_node" "test" {
  name      = %[1]q
  mesh_name = aws_appmesh_mesh.test.id

  spec {
    backend {
      virtual_service {
        virtual_service_name = "servicea.simpleapp.local"
      }
    }

    listener {
      port_mapping {
        port     = 8080
        protocol = "http"
      }

      outlier_detection {
        base_ejection_duration {
          unit  = "s"
          value = 6
        }

        interval {
          unit  = "ms"
          value = 10000
        }

        max_ejection_percent = 60
        max_server_errors    = 6
      }
    }

    service_discovery {
      dns {
        hostname = "serviceb.simpleapp.local"
      }
    }
  }
}
`, vnName))
}

func testAccAppmeshVirtualNodeConfig_listenerTimeout(meshName, vnName string) string {
	return acctest.ConfigCompose(testAccAppmeshVirtualNodeConfig_mesh(meshName), fmt.Sprintf(`
resource "aws_appmesh_virtual_node" "test" {
  name      = %[1]q
  mesh_name = aws_appmesh_mesh.test.id

  spec {
    backend {
      virtual_service {
        virtual_service_name = "servicea.simpleapp.local"
      }
    }

    listener {
      port_mapping {
        port     = 8080
        protocol = "tcp"
      }

      timeout {
        tcp {
          idle {
            unit  = "ms"
            value = 250000
          }
        }
      }
    }

    service_discovery {
      dns {
        hostname = "serviceb.simpleapp.local"
      }
    }
  }
}
`, vnName))
}

func testAccAppmeshVirtualNodeConfig_listenerTimeoutUpdated(meshName, vnName string) string {
	return acctest.ConfigCompose(testAccAppmeshVirtualNodeConfig_mesh(meshName), fmt.Sprintf(`
resource "aws_appmesh_virtual_node" "test" {
  name      = %[1]q
  mesh_name = aws_appmesh_mesh.test.id

  spec {
    backend {
      virtual_service {
        virtual_service_name = "servicea.simpleapp.local"
      }
    }

    listener {
      port_mapping {
        port     = 8080
        protocol = "http"
      }

      timeout {
        http {
          idle {
            unit  = "s"
            value = 10
          }

          per_request {
            unit  = "s"
            value = 5
          }
        }
      }
    }

    service_discovery {
      dns {
        hostname = "serviceb.simpleapp.local"
      }
    }
  }
}
`, vnName))
}

func testAccAppmeshVirtualNodeConfig_listenerTlsFile(meshName, vnName string) string {
	return acctest.ConfigCompose(testAccAppmeshVirtualNodeConfig_mesh(meshName), fmt.Sprintf(`
resource "aws_appmesh_virtual_node" "test" {
  name      = %[1]q
  mesh_name = aws_appmesh_mesh.test.id

  spec {
    backend {
      virtual_service {
        virtual_service_name = "servicea.simpleapp.local"
      }
    }

    listener {
      port_mapping {
        port     = 8080
        protocol = "http"
      }

      tls {
        certificate {
          file {
            certificate_chain = "/cert_chain.pem"
            private_key       = "/key.pem"
          }
        }

        mode = "PERMISSIVE"
      }
    }

    service_discovery {
      dns {
        hostname = "serviceb.simpleapp.local"
      }
    }
  }
}
`, vnName))
}

func testAccAppmeshVirtualNodeConfig_listenerTlsAcm(meshName, vnName, domain string) string {
	return acctest.ConfigCompose(
		testAccAppmeshVirtualNodeConfigRootCA(domain),
		testAccAppmeshVirtualNodeConfigPrivateCert(domain),
		testAccAppmeshVirtualNodeConfig_mesh(meshName),
		fmt.Sprintf(`
resource "aws_appmesh_virtual_node" "test" {
  name      = %[1]q
  mesh_name = aws_appmesh_mesh.test.id

  spec {
    backend {
      virtual_service {
        virtual_service_name = "servicea.simpleapp.local"
      }
    }

    listener {
      port_mapping {
        port     = 8080
        protocol = "http"
      }

      tls {
        certificate {
          acm {
            certificate_arn = aws_acm_certificate.test.arn
          }
        }

        mode = "STRICT"
      }
    }

    service_discovery {
      dns {
        hostname = "serviceb.simpleapp.local"
      }
    }
  }
}
`, vnName))
}

func testAccAppmeshVirtualNodeConfig_listenerValidation(meshName, vnName string) string {
	return acctest.ConfigCompose(testAccAppmeshVirtualNodeConfig_mesh(meshName), fmt.Sprintf(`
resource "aws_appmesh_virtual_node" "test" {
  name      = %[1]q
  mesh_name = aws_appmesh_mesh.test.id

  spec {
    backend {
      virtual_service {
        virtual_service_name = "servicea.simpleapp.local"
      }
    }

    listener {
      port_mapping {
        port     = 8080
        protocol = "http"
      }

      tls {
        certificate {
          sds {
            secret_name = "very-secret"
          }
        }

        mode = "PERMISSIVE"

        validation {
          subject_alternative_names {
            match {
              exact = ["abc.example.com", "xyz.example.com"]
            }
          }

          trust {
            file {
              certificate_chain = "/cert_chain.pem"
            }
          }
        }
      }
    }

    service_discovery {
      dns {
        hostname = "serviceb.simpleapp.local"
      }
    }
  }
}
`, vnName))
}

func testAccAppmeshVirtualNodeConfig_listenerValidationUpdated(meshName, vnName string) string {
	return acctest.ConfigCompose(testAccAppmeshVirtualNodeConfig_mesh(meshName), fmt.Sprintf(`
resource "aws_appmesh_virtual_node" "test" {
  name      = %[1]q
  mesh_name = aws_appmesh_mesh.test.id

  spec {
    backend {
      virtual_service {
        virtual_service_name = "servicea.simpleapp.local"
      }
    }

    listener {
      port_mapping {
        port     = 8080
        protocol = "http"
      }

      tls {
        certificate {
          sds {
            secret_name = "top-secret"
          }
        }

        mode = "PERMISSIVE"

        validation {
          trust {
            sds {
              secret_name = "confidential"
            }
          }
        }
      }
    }

    service_discovery {
      dns {
        hostname = "serviceb.simpleapp.local"
      }
    }
  }
}
`, vnName))
}

func testAccAppmeshVirtualNodeConfig_logging(meshName, vnName, path string) string {
	return acctest.ConfigCompose(testAccAppmeshVirtualNodeConfig_mesh(meshName), fmt.Sprintf(`
resource "aws_appmesh_virtual_node" "test" {
  name      = %[1]q
  mesh_name = aws_appmesh_mesh.test.id

  spec {
    backend {
      virtual_service {
        virtual_service_name = "servicea.simpleapp.local"
      }
    }

    listener {
      port_mapping {
        port     = 8080
        protocol = "http"
      }
    }

    logging {
      access_log {
        file {
          path = %[2]q
        }
      }
    }

    service_discovery {
      dns {
        hostname = "serviceb.simpleapp.local"
      }
    }
  }
}
`, vnName, path))
}

func testAccAppmeshVirtualNodeConfig_tags(meshName, vnName, tagKey1, tagValue1, tagKey2, tagValue2 string) string {
	return acctest.ConfigCompose(testAccAppmeshVirtualNodeConfig_mesh(meshName), fmt.Sprintf(`
resource "aws_appmesh_virtual_node" "test" {
  name      = %[1]q
  mesh_name = aws_appmesh_mesh.test.id

  spec {}

  tags = {
    %[2]s = %[3]q
    %[4]s = %[5]q
  }
}
`, vnName, tagKey1, tagValue1, tagKey2, tagValue2))
}
