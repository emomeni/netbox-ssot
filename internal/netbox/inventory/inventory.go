package inventory

import (
	"fmt"
	"time"

	"github.com/bl4ko/netbox-ssot/internal/logger"
	"github.com/bl4ko/netbox-ssot/internal/netbox/objects"
	"github.com/bl4ko/netbox-ssot/internal/netbox/service"
	"github.com/bl4ko/netbox-ssot/internal/parser"
	"github.com/bl4ko/netbox-ssot/internal/utils"
)

// NetBoxInventory is a singleton class to manage a inventory of NetBoxObject objects
type NetBoxInventory struct {
	// Logger is the logger used for logging messages
	Logger *logger.Logger
	// NetboxConfig is the Netbox configuration
	NetboxConfig *parser.NetboxConfig
	// NetboxApi is the Netbox API object, for communicating with the Netbox API
	NetboxApi *service.NetboxAPI
	// Tags is a list of all tags in the netbox inventory
	Tags []*objects.Tag
	// ContactGroupsIndexByName is a map of all contact groups indexed by their names.
	ContactGroupsIndexByName map[string]*objects.ContactGroup
	// ContactRolesIndexByName is a map of all contact roles indexed by their names.
	ContactRolesIndexByName map[string]*objects.ContactRole
	// ContactsIndexByName is a map of all contacts in the Netbox's inventory, indexed by their names
	ContactsIndexByName map[string]*objects.Contact
	// ContactAssignmentsIndexByContentTypeAndObjectIdAndContactIdAndRoleId is a map of all contact assignments indexed by their content type, object id, contact id and role id.
	ContactAssignmentsIndexByContentTypeAndObjectIdAndContactIdAndRoleId map[string]map[int]map[int]map[int]*objects.ContactAssignment
	// SitesIndexByName is a map of all sites in the Netbox's inventory, indexed by their name
	SitesIndexByName map[string]*objects.Site
	// ManufacturersIndexByName is a map of all manufacturers in the Netbox's inventory, indexed by their name
	ManufacturersIndexByName map[string]*objects.Manufacturer
	// PlatformsIndexByName is a map of all platforms in the Netbox's inventory, indexed by their name
	PlatformsIndexByName map[string]*objects.Platform
	// TenantsIndexByName is a map of all tenants in the Netbox's inventory, indexed by their name
	TenantsIndexByName map[string]*objects.Tenant
	// DeviceTypesIndexByModel is a map of all device types in the Netbox's inventory, indexed by their model
	DeviceTypesIndexByModel map[string]*objects.DeviceType
	// DevicesIndexByNameAndSiteId is a map of all devices in the Netbox's inventory, indexed by their name, and
	// site ID (This is because, netbox constraints: https://github.com/netbox-community/netbox/blob/3d941411d438f77b66d2036edf690c14b459af58/netbox/dcim/models/devices.py#L775)
	DevicesIndexByNameAndSiteId map[string]map[int]*objects.Device
	// VlanGroupsIndexByName is a map of all VlanGroups in the Netbox's inventory, indexed by their name
	VlanGroupsIndexByName map[string]*objects.VlanGroup
	// VlansIndexByVlanGroupIdAndVid is a map of all vlans in the Netbox's inventory, indexed by their VlanGroup and vid.
	VlansIndexByVlanGroupIdAndVid map[int]map[int]*objects.Vlan
	// ClusterGroupsIndexByName is a map of all cluster groups in the Netbox's inventory, indexed by their name
	ClusterGroupsIndexByName map[string]*objects.ClusterGroup
	// ClusterTypesIndexByName is a map of all cluster types in the Netbox's inventory, indexed by their name
	ClusterTypesIndexByName map[string]*objects.ClusterType
	// ClustersIndexByName is a map of all clusters in the Netbox's inventory, indexed by their name
	ClustersIndexByName map[string]*objects.Cluster
	// Netbox's Device Roles is a map of all device roles in the inventory, indexed by name
	DeviceRolesIndexByName map[string]*objects.DeviceRole
	// CustomFieldsIndexByName is a map of all custom fields in the inventory, indexed by name
	CustomFieldsIndexByName map[string]*objects.CustomField
	// InterfacesIndexByDeviceAnName is a map of all interfaces in the inventory, indexed by their's
	// device id and their name.
	InterfacesIndexByDeviceIdAndName map[int]map[string]*objects.Interface
	// VirtualMachinedIndexByName is a map of all virtual machines in the inventory, indexed by their name
	VMsIndexByName map[string]*objects.VM
	// VirtualMachineInterfacesIndexByVMAndName is a map of all virtual machine interfaces in the inventory, indexed by their's virtual machine id and their name
	VMInterfacesIndexByVMIdAndName map[int]map[string]*objects.VMInterface
	// IPAdressesIndexByAddress is a map of all IP addresses in the inventory, indexed by their address
	IPAdressesIndexByAddress map[string]*objects.IPAddress

	// Orphan manager is a map of objectAPIPath to a set of managed ids for that object type.
	//
	// {
	//		service.DeviceApiPath: {22: true, 3: true, ...},
	//		"/api/dcim/interface/": {15: true, 36: true, ...},
	//  	service.Clusters: {121: true, 122: true, ...},
	//  	"...": [...]
	// }
	//
	// It stores which objects have been created by netbox-ssot and can be deleted
	// because they are not available in the sources anymore
	OrphanManager map[string]map[int]bool

	// OrphanObjectPriority is a map that stores priorities for each object. This is necessary
	// because map order is non deterministic and if we delete dependent object first we will
	// get the dependency error.
	//
	// {
	//   0: service.TagApiPath
	//   1: "/api/extras/custom-fields/"
	//
	// }
	OrphanObjectPriority map[int]string

	// Tag used by netbox-ssot to mark devices that are managed by it
	SsotTag *objects.Tag
}

// Func string representation
func (nbi NetBoxInventory) String() string {
	return fmt.Sprintf("NetBoxInventory{Logger: %+v, NetboxConfig: %+v...}", nbi.Logger, nbi.NetboxConfig)
}

// NewNetboxInventory creates a new NetBoxInventory object.
// It takes a logger and a NetboxConfig as parameters, and returns a pointer to the newly created NetBoxInventory.
// The logger is used for logging messages, and the NetboxConfig is used to configure the NetBoxInventory.
func NewNetboxInventory(logger *logger.Logger, nbConfig *parser.NetboxConfig) *NetBoxInventory {
	// Starts with 0 for easier integration with for loops
	orphanObjectPriority := map[int]string{
		0:  service.VlanGroupApiPath,
		1:  service.VlanApiPath,
		2:  service.IpAddressApiPath,
		3:  service.InterfaceApiPath,
		4:  service.VMInterfaceApiPath,
		5:  service.VirtualMachineApiPath,
		6:  service.DeviceApiPath,
		7:  service.PlatformApiPath,
		8:  service.DeviceTypeApiPath,
		9:  service.ManufacturerApiPath,
		10: service.DeviceRoleApiPath,
		11: service.ClusterApiPath,
		12: service.ClusterTypeApiPath,
		13: service.ClusterGroupApiPath,
		14: service.ContactApiPath,
		15: service.ContactAssignmentApiPath,
	}
	nbi := &NetBoxInventory{Logger: logger, NetboxConfig: nbConfig, OrphanManager: make(map[string]map[int]bool), OrphanObjectPriority: orphanObjectPriority}
	return nbi
}

// Init function that initializes the NetBoxInventory object with objects from Netbox
func (nbi *NetBoxInventory) Init() error {
	baseURL := fmt.Sprintf("%s://%s:%d", nbi.NetboxConfig.HTTPScheme, nbi.NetboxConfig.Hostname, nbi.NetboxConfig.Port)

	nbi.Logger.Debug("Initializing Netbox API with baseURL: ", baseURL)
	nbi.NetboxApi = service.NewNetBoxAPI(nbi.Logger, baseURL, nbi.NetboxConfig.ApiToken, nbi.NetboxConfig.ValidateCert, nbi.NetboxConfig.Timeout)

	// order matters. TODO: use parallelization in the future, on the init functions that can be parallelized
	initFunctions := []func() error{
		nbi.InitTags,
		nbi.InitContactGroups,
		nbi.InitContactRoles,
		nbi.InitAdminContactRole,
		nbi.InitContacts,
		nbi.InitContactAssignments,
		nbi.InitTenants,
		nbi.InitSites,
		nbi.InitManufacturers,
		nbi.InitPlatforms,
		nbi.InitDevices,
		nbi.InitInterfaces,
		nbi.InitIPAddresses,
		nbi.InitVlanGroups,
		nbi.InitDefaultVlanGroup,
		nbi.InitVlans,
		nbi.InitDeviceRoles,
		nbi.InitServerDeviceRole,
		nbi.InitDeviceTypes,
		nbi.InitCustomFields,
		nbi.InitSsotCustomFields,
		nbi.InitClusterGroups,
		nbi.InitClusterTypes,
		nbi.InitClusters,
		nbi.InitVMs,
		nbi.InitVMInterfaces,
	}
	for _, initFunc := range initFunctions {
		startTime := time.Now()
		if err := initFunc(); err != nil {
			return fmt.Errorf("%s: %s", err, utils.ExtractFunctionName(initFunc))
		}
		duration := time.Since(startTime)
		nbi.Logger.Infof("Successfully initialized %s in %f seconds", utils.ExtractFunctionName(initFunc), duration.Seconds())
	}

	return nil

}
