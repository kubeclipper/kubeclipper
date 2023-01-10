package metallb

const ipAddressPoolTemplate = `
apiVersion: metallb.io/v1beta1
kind: IPAddressPool
metadata:
  name: first-pool
  namespace: metallb-system
spec:
  addresses:
  {{- range .Addresses }}
  - {{ . }}
  {{- end }}
`

const advertisementTemplate = `
apiVersion: metallb.io/v1beta1
kind: {{.Mode}}Advertisement
metadata:
  name: local
  namespace: metallb-system
`

const manifestTemplate = `
apiVersion: v1
kind: Namespace
metadata:
  labels:
    pod-security.kubernetes.io/audit: privileged
    pod-security.kubernetes.io/enforce: privileged
    pod-security.kubernetes.io/warn: privileged
  name: metallb-system
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  annotations:
    controller-gen.kubebuilder.io/version: v0.7.0
  name: addresspools.metallb.io
spec:
  conversion:
    strategy: Webhook
    webhook:
      clientConfig:
        caBundle: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tDQpNSUlGWlRDQ0EwMmdBd0lCQWdJVU5GRW1XcTM3MVpKdGkrMmlSQzk1WmpBV1MxZ3dEUVlKS29aSWh2Y05BUUVMDQpCUUF3UWpFTE1Ba0dBMVVFQmhNQ1dGZ3hGVEFUQmdOVkJBY01ERVJsWm1GMWJIUWdRMmwwZVRFY01Cb0dBMVVFDQpDZ3dUUkdWbVlYVnNkQ0JEYjIxd1lXNTVJRXgwWkRBZUZ3MHlNakEzTVRrd09UTXlNek5hRncweU1qQTRNVGd3DQpPVE15TXpOYU1FSXhDekFKQmdOVkJBWVRBbGhZTVJVd0V3WURWUVFIREF4RVpXWmhkV3gwSUVOcGRIa3hIREFhDQpCZ05WQkFvTUUwUmxabUYxYkhRZ1EyOXRjR0Z1ZVNCTWRHUXdnZ0lpTUEwR0NTcUdTSWIzRFFFQkFRVUFBNElDDQpEd0F3Z2dJS0FvSUNBUUNxVFpxMWZRcC9vYkdlenhES0o3OVB3Ny94azJwellualNzMlkzb1ZYSm5sRmM4YjVlDQpma2ZZQnY2bndscW1keW5PL2phWFBaQmRQSS82aFdOUDBkdVhadEtWU0NCUUpyZzEyOGNXb3F0MGNTN3pLb1VpDQpvcU1tQ0QvRXVBeFFNZjhRZDF2c1gvVllkZ0poVTZBRXJLZEpIaXpFOUJtUkNkTDBGMW1OVW55Rk82UnRtWFZUDQpidkxsTDVYeTc2R0FaQVBLOFB4aVlDa0NtbDdxN0VnTWNiOXlLWldCYmlxQ3VkTXE5TGJLNmdKNzF6YkZnSXV4DQo1L1pXK2JraTB2RlplWk9ZODUxb1psckFUNzJvMDI4NHNTWW9uN0pHZVZkY3NoUnh5R1VpSFpSTzdkaXZVTDVTDQpmM2JmSDFYbWY1ZDQzT0NWTWRuUUV2NWVaOG8zeWVLa3ZrbkZQUGVJMU9BbjdGbDlFRVNNR2dhOGFaSG1URSttDQpsLzlMSmdDYjBnQmtPT0M0WnV4bWh2aERKV1EzWnJCS3pMQlNUZXN0NWlLNVlwcXRWVVk2THRyRW9FelVTK1lsDQpwWndXY2VQWHlHeHM5ZURsR3lNVmQraW15Y3NTU1UvVno2Mmx6MnZCS21NTXBkYldDQWhud0RsRTVqU2dyMjRRDQp0eGNXLys2N3d5KzhuQlI3UXdqVTFITndVRjBzeERWdEwrZ1NHVERnSEVZSlhZelYvT05zMy94TkpoVFNPSkxNDQpoeXNVdyttaGdackdhbUdXcHVIVU1DUitvTWJzMTc1UkcrQjJnUFFHVytPTjJnUTRyOXN2b0ZBNHBBQm8xd1dLDQpRYjRhY3pmeVVscElBOVFoSmFsZEY3S3dPSHVlV3gwRUNrNXg0T2tvVDBvWVp0dzFiR0JjRGtaSmF3SURBUUFCDQpvMU13VVRBZEJnTlZIUTRFRmdRVW90UlNIUm9IWTEyRFZ4R0NCdEhpb1g2ZmVFQXdId1lEVlIwakJCZ3dGb0FVDQpvdFJTSFJvSFkxMkRWeEdDQnRIaW9YNmZlRUF3RHdZRFZSMFRBUUgvQkFVd0F3RUIvekFOQmdrcWhraUc5dzBCDQpBUXNGQUFPQ0FnRUFSbkpsWWRjMTFHd0VxWnh6RDF2R3BDR2pDN2VWTlQ3aVY1d3IybXlybHdPYi9aUWFEa0xYDQpvVStaOVVXT1VlSXJTdzUydDdmQUpvVVAwSm5iYkMveVIrU1lqUGhvUXNiVHduOTc2ZldBWTduM3FMOXhCd1Y0DQphek41OXNjeUp0dlhMeUtOL2N5ak1ReDRLajBIMFg0bWJ6bzVZNUtzWWtYVU0vOEFPdWZMcEd0S1NGVGgrSEFDDQpab1Q5YnZHS25adnNHd0tYZFF0Wnh0akhaUjVqK3U3ZGtQOTJBT051RFNabS8rWVV4b2tBK09JbzdSR3BwSHNXDQo1ZTdNY0FTVXRtb1FORXd6dVFoVkJaRWQ1OGtKYjUrV0VWbGNzanlXNnRTbzErZ25tTWNqR1BsMWgxR2hVbjV4DQpFY0lWRnBIWXM5YWo1NmpBSjk1MVQvZjhMaWxmTlVnanBLQ0c1bnl0SUt3emxhOHNtdGlPdm1UNEpYbXBwSkI2DQo4bmdHRVluVjUrUTYwWFJ2OEhSSGp1VG9CRHVhaERrVDA2R1JGODU1d09FR2V4bkZpMXZYWUxLVllWb1V2MXRKDQo4dVdUR1pwNllDSVJldlBqbzg5ZytWTlJSaVFYUThJd0dybXE5c0RoVTlqTjA0SjdVL1RvRDFpNHE3VnlsRUc5DQorV1VGNkNLaEdBeTJIaEhwVncyTGFoOS9lUzdZMUZ1YURrWmhPZG1laG1BOCtqdHNZamJadnR5Mm1SWlF0UUZzDQpUU1VUUjREbUR2bVVPRVRmeStpRHdzK2RkWXVNTnJGeVVYV2dkMnpBQU4ydVl1UHFGY2pRcFNPODFzVTJTU3R3DQoxVzAyeUtYOGJEYmZFdjBzbUh3UzliQnFlSGo5NEM1Mjg0YXpsdTBmaUdpTm1OUEM4ckJLRmhBPQ0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQ==
        service:
          name: webhook-service
          namespace: metallb-system
          path: /convert
      conversionReviewVersions:
      - v1alpha1
      - v1beta1
  group: metallb.io
  names:
    kind: AddressPool
    listKind: AddressPoolList
    plural: addresspools
    singular: addresspool
  scope: Namespaced
  versions:
  - deprecated: true
    deprecationWarning: metallb.io v1alpha1 AddressPool is deprecated
    name: v1alpha1
    schema:
      openAPIV3Schema:
        description: AddressPool is the Schema for the addresspools API.
        properties:
          apiVersion:
            description: 'APIVersion defines the versioned schema of this representation
              of an object. Servers should convert recognized schemas to the latest
              internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
            type: string
          kind:
            description: 'Kind is a string value representing the REST resource this
              object represents. Servers may infer this from the endpoint the client
              submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
            type: string
          metadata:
            type: object
          spec:
            description: AddressPoolSpec defines the desired state of AddressPool.
            properties:
              addresses:
                description: A list of IP address ranges over which MetalLB has authority.
                  You can list multiple ranges in a single pool, they will all share
                  the same settings. Each range can be either a CIDR prefix, or an
                  explicit start-end range of IPs.
                items:
                  type: string
                type: array
              autoAssign:
                default: true
                description: AutoAssign flag used to prevent MetallB from automatic
                  allocation for a pool.
                type: boolean
              bgpAdvertisements:
                description: When an IP is allocated from this pool, how should it
                  be translated into BGP announcements?
                items:
                  properties:
                    aggregationLength:
                      default: 32
                      description: The aggregation-length advertisement option lets
                        you “roll up” the /32s into a larger prefix.
                      format: int32
                      minimum: 1
                      type: integer
                    aggregationLengthV6:
                      default: 128
                      description: Optional, defaults to 128 (i.e. no aggregation)
                        if not specified.
                      format: int32
                      type: integer
                    communities:
                      description: BGP communities
                      items:
                        type: string
                      type: array
                    localPref:
                      description: BGP LOCAL_PREF attribute which is used by BGP best
                        path algorithm, Path with higher localpref is preferred over
                        one with lower localpref.
                      format: int32
                      type: integer
                  type: object
                type: array
              protocol:
                description: Protocol can be used to select how the announcement is
                  done.
                enum:
                - layer2
                - bgp
                type: string
            required:
            - addresses
            - protocol
            type: object
          status:
            description: AddressPoolStatus defines the observed state of AddressPool.
            type: object
        required:
        - spec
        type: object
    served: true
    storage: false
    subresources:
      status: {}
  - deprecated: true
    deprecationWarning: metallb.io v1beta1 AddressPool is deprecated, consider using
      IPAddressPool
    name: v1beta1
    schema:
      openAPIV3Schema:
        description: AddressPool represents a pool of IP addresses that can be allocated
          to LoadBalancer services. AddressPool is deprecated and being replaced by
          IPAddressPool.
        properties:
          apiVersion:
            description: 'APIVersion defines the versioned schema of this representation
              of an object. Servers should convert recognized schemas to the latest
              internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
            type: string
          kind:
            description: 'Kind is a string value representing the REST resource this
              object represents. Servers may infer this from the endpoint the client
              submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
            type: string
          metadata:
            type: object
          spec:
            description: AddressPoolSpec defines the desired state of AddressPool.
            properties:
              addresses:
                description: A list of IP address ranges over which MetalLB has authority.
                  You can list multiple ranges in a single pool, they will all share
                  the same settings. Each range can be either a CIDR prefix, or an
                  explicit start-end range of IPs.
                items:
                  type: string
                type: array
              autoAssign:
                default: true
                description: AutoAssign flag used to prevent MetallB from automatic
                  allocation for a pool.
                type: boolean
              bgpAdvertisements:
                description: Drives how an IP allocated from this pool should translated
                  into BGP announcements.
                items:
                  properties:
                    aggregationLength:
                      default: 32
                      description: The aggregation-length advertisement option lets
                        you “roll up” the /32s into a larger prefix.
                      format: int32
                      minimum: 1
                      type: integer
                    aggregationLengthV6:
                      default: 128
                      description: Optional, defaults to 128 (i.e. no aggregation)
                        if not specified.
                      format: int32
                      type: integer
                    communities:
                      description: BGP communities to be associated with the given
                        advertisement.
                      items:
                        type: string
                      type: array
                    localPref:
                      description: BGP LOCAL_PREF attribute which is used by BGP best
                        path algorithm, Path with higher localpref is preferred over
                        one with lower localpref.
                      format: int32
                      type: integer
                  type: object
                type: array
              protocol:
                description: Protocol can be used to select how the announcement is
                  done.
                enum:
                - layer2
                - bgp
                type: string
            required:
            - addresses
            - protocol
            type: object
          status:
            description: AddressPoolStatus defines the observed state of AddressPool.
            type: object
        required:
        - spec
        type: object
    served: true
    storage: true
    subresources:
      status: {}
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: []
  storedVersions: []
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  annotations:
    controller-gen.kubebuilder.io/version: v0.7.0
  creationTimestamp: null
  name: bfdprofiles.metallb.io
spec:
  group: metallb.io
  names:
    kind: BFDProfile
    listKind: BFDProfileList
    plural: bfdprofiles
    singular: bfdprofile
  scope: Namespaced
  versions:
  - additionalPrinterColumns:
    - jsonPath: .spec.passiveMode
      name: Passive Mode
      type: boolean
    - jsonPath: .spec.transmitInterval
      name: Transmit Interval
      type: integer
    - jsonPath: .spec.receiveInterval
      name: Receive Interval
      type: integer
    - jsonPath: .spec.detectMultiplier
      name: Multiplier
      type: integer
    name: v1beta1
    schema:
      openAPIV3Schema:
        description: BFDProfile represents the settings of the bfd session that can
          be optionally associated with a BGP session.
        properties:
          apiVersion:
            description: 'APIVersion defines the versioned schema of this representation
              of an object. Servers should convert recognized schemas to the latest
              internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
            type: string
          kind:
            description: 'Kind is a string value representing the REST resource this
              object represents. Servers may infer this from the endpoint the client
              submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
            type: string
          metadata:
            type: object
          spec:
            description: BFDProfileSpec defines the desired state of BFDProfile.
            properties:
              detectMultiplier:
                description: Configures the detection multiplier to determine packet
                  loss. The remote transmission interval will be multiplied by this
                  value to determine the connection loss detection timer.
                format: int32
                maximum: 255
                minimum: 2
                type: integer
              echoInterval:
                description: Configures the minimal echo receive transmission interval
                  that this system is capable of handling in milliseconds. Defaults
                  to 50ms
                format: int32
                maximum: 60000
                minimum: 10
                type: integer
              echoMode:
                description: Enables or disables the echo transmission mode. This
                  mode is disabled by default, and not supported on multi hops setups.
                type: boolean
              minimumTtl:
                description: 'For multi hop sessions only: configure the minimum expected
                  TTL for an incoming BFD control packet.'
                format: int32
                maximum: 254
                minimum: 1
                type: integer
              passiveMode:
                description: 'Mark session as passive: a passive session will not
                  attempt to start the connection and will wait for control packets
                  from peer before it begins replying.'
                type: boolean
              receiveInterval:
                description: The minimum interval that this system is capable of receiving
                  control packets in milliseconds. Defaults to 300ms.
                format: int32
                maximum: 60000
                minimum: 10
                type: integer
              transmitInterval:
                description: The minimum transmission interval (less jitter) that
                  this system wants to use to send BFD control packets in milliseconds.
                  Defaults to 300ms
                format: int32
                maximum: 60000
                minimum: 10
                type: integer
            type: object
          status:
            description: BFDProfileStatus defines the observed state of BFDProfile.
            type: object
        type: object
    served: true
    storage: true
    subresources:
      status: {}
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: []
  storedVersions: []
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  annotations:
    controller-gen.kubebuilder.io/version: v0.7.0
  creationTimestamp: null
  name: bgpadvertisements.metallb.io
spec:
  group: metallb.io
  names:
    kind: BGPAdvertisement
    listKind: BGPAdvertisementList
    plural: bgpadvertisements
    singular: bgpadvertisement
  scope: Namespaced
  versions:
  - additionalPrinterColumns:
    - jsonPath: .spec.ipAddressPools
      name: IPAddressPools
      type: string
    - jsonPath: .spec.ipAddressPoolSelectors
      name: IPAddressPool Selectors
      type: string
    - jsonPath: .spec.peers
      name: Peers
      type: string
    - jsonPath: .spec.nodeSelectors
      name: Node Selectors
      priority: 10
      type: string
    name: v1beta1
    schema:
      openAPIV3Schema:
        description: BGPAdvertisement allows to advertise the IPs coming from the
          selected IPAddressPools via BGP, setting the parameters of the BGP Advertisement.
        properties:
          apiVersion:
            description: 'APIVersion defines the versioned schema of this representation
              of an object. Servers should convert recognized schemas to the latest
              internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
            type: string
          kind:
            description: 'Kind is a string value representing the REST resource this
              object represents. Servers may infer this from the endpoint the client
              submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
            type: string
          metadata:
            type: object
          spec:
            description: BGPAdvertisementSpec defines the desired state of BGPAdvertisement.
            properties:
              aggregationLength:
                default: 32
                description: The aggregation-length advertisement option lets you
                  “roll up” the /32s into a larger prefix. Defaults to 32. Works for
                  IPv4 addresses.
                format: int32
                minimum: 1
                type: integer
              aggregationLengthV6:
                default: 128
                description: The aggregation-length advertisement option lets you
                  “roll up” the /128s into a larger prefix. Defaults to 128. Works
                  for IPv6 addresses.
                format: int32
                type: integer
              communities:
                description: The BGP communities to be associated with the announcement.
                  Each item can be a community of the form 1234:1234 or the name of
                  an alias defined in the Community CRD.
                items:
                  type: string
                type: array
              ipAddressPoolSelectors:
                description: A selector for the IPAddressPools which would get advertised
                  via this advertisement. If no IPAddressPool is selected by this
                  or by the list, the advertisement is applied to all the IPAddressPools.
                items:
                  description: A label selector is a label query over a set of resources.
                    The result of matchLabels and matchExpressions are ANDed. An empty
                    label selector matches all objects. A null label selector matches
                    no objects.
                  properties:
                    matchExpressions:
                      description: matchExpressions is a list of label selector requirements.
                        The requirements are ANDed.
                      items:
                        description: A label selector requirement is a selector that
                          contains values, a key, and an operator that relates the
                          key and values.
                        properties:
                          key:
                            description: key is the label key that the selector applies
                              to.
                            type: string
                          operator:
                            description: operator represents a key's relationship
                              to a set of values. Valid operators are In, NotIn, Exists
                              and DoesNotExist.
                            type: string
                          values:
                            description: values is an array of string values. If the
                              operator is In or NotIn, the values array must be non-empty.
                              If the operator is Exists or DoesNotExist, the values
                              array must be empty. This array is replaced during a
                              strategic merge patch.
                            items:
                              type: string
                            type: array
                        required:
                        - key
                        - operator
                        type: object
                      type: array
                    matchLabels:
                      additionalProperties:
                        type: string
                      description: matchLabels is a map of {key,value} pairs. A single
                        {key,value} in the matchLabels map is equivalent to an element
                        of matchExpressions, whose key field is "key", the operator
                        is "In", and the values array contains only "value". The requirements
                        are ANDed.
                      type: object
                  type: object
                type: array
              ipAddressPools:
                description: The list of IPAddressPools to advertise via this advertisement,
                  selected by name.
                items:
                  type: string
                type: array
              localPref:
                description: The BGP LOCAL_PREF attribute which is used by BGP best
                  path algorithm, Path with higher localpref is preferred over one
                  with lower localpref.
                format: int32
                type: integer
              nodeSelectors:
                description: NodeSelectors allows to limit the nodes to announce as
                  next hops for the LoadBalancer IP. When empty, all the nodes having  are
                  announced as next hops.
                items:
                  description: A label selector is a label query over a set of resources.
                    The result of matchLabels and matchExpressions are ANDed. An empty
                    label selector matches all objects. A null label selector matches
                    no objects.
                  properties:
                    matchExpressions:
                      description: matchExpressions is a list of label selector requirements.
                        The requirements are ANDed.
                      items:
                        description: A label selector requirement is a selector that
                          contains values, a key, and an operator that relates the
                          key and values.
                        properties:
                          key:
                            description: key is the label key that the selector applies
                              to.
                            type: string
                          operator:
                            description: operator represents a key's relationship
                              to a set of values. Valid operators are In, NotIn, Exists
                              and DoesNotExist.
                            type: string
                          values:
                            description: values is an array of string values. If the
                              operator is In or NotIn, the values array must be non-empty.
                              If the operator is Exists or DoesNotExist, the values
                              array must be empty. This array is replaced during a
                              strategic merge patch.
                            items:
                              type: string
                            type: array
                        required:
                        - key
                        - operator
                        type: object
                      type: array
                    matchLabels:
                      additionalProperties:
                        type: string
                      description: matchLabels is a map of {key,value} pairs. A single
                        {key,value} in the matchLabels map is equivalent to an element
                        of matchExpressions, whose key field is "key", the operator
                        is "In", and the values array contains only "value". The requirements
                        are ANDed.
                      type: object
                  type: object
                type: array
              peers:
                description: Peers limits the bgppeer to advertise the ips of the
                  selected pools to. When empty, the loadbalancer IP is announced
                  to all the BGPPeers configured.
                items:
                  type: string
                type: array
            type: object
          status:
            description: BGPAdvertisementStatus defines the observed state of BGPAdvertisement.
            type: object
        type: object
    served: true
    storage: true
    subresources:
      status: {}
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: []
  storedVersions: []
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  annotations:
    controller-gen.kubebuilder.io/version: v0.7.0
  name: bgppeers.metallb.io
spec:
  conversion:
    strategy: Webhook
    webhook:
      clientConfig:
        caBundle: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tDQpNSUlGWlRDQ0EwMmdBd0lCQWdJVU5GRW1XcTM3MVpKdGkrMmlSQzk1WmpBV1MxZ3dEUVlKS29aSWh2Y05BUUVMDQpCUUF3UWpFTE1Ba0dBMVVFQmhNQ1dGZ3hGVEFUQmdOVkJBY01ERVJsWm1GMWJIUWdRMmwwZVRFY01Cb0dBMVVFDQpDZ3dUUkdWbVlYVnNkQ0JEYjIxd1lXNTVJRXgwWkRBZUZ3MHlNakEzTVRrd09UTXlNek5hRncweU1qQTRNVGd3DQpPVE15TXpOYU1FSXhDekFKQmdOVkJBWVRBbGhZTVJVd0V3WURWUVFIREF4RVpXWmhkV3gwSUVOcGRIa3hIREFhDQpCZ05WQkFvTUUwUmxabUYxYkhRZ1EyOXRjR0Z1ZVNCTWRHUXdnZ0lpTUEwR0NTcUdTSWIzRFFFQkFRVUFBNElDDQpEd0F3Z2dJS0FvSUNBUUNxVFpxMWZRcC9vYkdlenhES0o3OVB3Ny94azJwellualNzMlkzb1ZYSm5sRmM4YjVlDQpma2ZZQnY2bndscW1keW5PL2phWFBaQmRQSS82aFdOUDBkdVhadEtWU0NCUUpyZzEyOGNXb3F0MGNTN3pLb1VpDQpvcU1tQ0QvRXVBeFFNZjhRZDF2c1gvVllkZ0poVTZBRXJLZEpIaXpFOUJtUkNkTDBGMW1OVW55Rk82UnRtWFZUDQpidkxsTDVYeTc2R0FaQVBLOFB4aVlDa0NtbDdxN0VnTWNiOXlLWldCYmlxQ3VkTXE5TGJLNmdKNzF6YkZnSXV4DQo1L1pXK2JraTB2RlplWk9ZODUxb1psckFUNzJvMDI4NHNTWW9uN0pHZVZkY3NoUnh5R1VpSFpSTzdkaXZVTDVTDQpmM2JmSDFYbWY1ZDQzT0NWTWRuUUV2NWVaOG8zeWVLa3ZrbkZQUGVJMU9BbjdGbDlFRVNNR2dhOGFaSG1URSttDQpsLzlMSmdDYjBnQmtPT0M0WnV4bWh2aERKV1EzWnJCS3pMQlNUZXN0NWlLNVlwcXRWVVk2THRyRW9FelVTK1lsDQpwWndXY2VQWHlHeHM5ZURsR3lNVmQraW15Y3NTU1UvVno2Mmx6MnZCS21NTXBkYldDQWhud0RsRTVqU2dyMjRRDQp0eGNXLys2N3d5KzhuQlI3UXdqVTFITndVRjBzeERWdEwrZ1NHVERnSEVZSlhZelYvT05zMy94TkpoVFNPSkxNDQpoeXNVdyttaGdackdhbUdXcHVIVU1DUitvTWJzMTc1UkcrQjJnUFFHVytPTjJnUTRyOXN2b0ZBNHBBQm8xd1dLDQpRYjRhY3pmeVVscElBOVFoSmFsZEY3S3dPSHVlV3gwRUNrNXg0T2tvVDBvWVp0dzFiR0JjRGtaSmF3SURBUUFCDQpvMU13VVRBZEJnTlZIUTRFRmdRVW90UlNIUm9IWTEyRFZ4R0NCdEhpb1g2ZmVFQXdId1lEVlIwakJCZ3dGb0FVDQpvdFJTSFJvSFkxMkRWeEdDQnRIaW9YNmZlRUF3RHdZRFZSMFRBUUgvQkFVd0F3RUIvekFOQmdrcWhraUc5dzBCDQpBUXNGQUFPQ0FnRUFSbkpsWWRjMTFHd0VxWnh6RDF2R3BDR2pDN2VWTlQ3aVY1d3IybXlybHdPYi9aUWFEa0xYDQpvVStaOVVXT1VlSXJTdzUydDdmQUpvVVAwSm5iYkMveVIrU1lqUGhvUXNiVHduOTc2ZldBWTduM3FMOXhCd1Y0DQphek41OXNjeUp0dlhMeUtOL2N5ak1ReDRLajBIMFg0bWJ6bzVZNUtzWWtYVU0vOEFPdWZMcEd0S1NGVGgrSEFDDQpab1Q5YnZHS25adnNHd0tYZFF0Wnh0akhaUjVqK3U3ZGtQOTJBT051RFNabS8rWVV4b2tBK09JbzdSR3BwSHNXDQo1ZTdNY0FTVXRtb1FORXd6dVFoVkJaRWQ1OGtKYjUrV0VWbGNzanlXNnRTbzErZ25tTWNqR1BsMWgxR2hVbjV4DQpFY0lWRnBIWXM5YWo1NmpBSjk1MVQvZjhMaWxmTlVnanBLQ0c1bnl0SUt3emxhOHNtdGlPdm1UNEpYbXBwSkI2DQo4bmdHRVluVjUrUTYwWFJ2OEhSSGp1VG9CRHVhaERrVDA2R1JGODU1d09FR2V4bkZpMXZYWUxLVllWb1V2MXRKDQo4dVdUR1pwNllDSVJldlBqbzg5ZytWTlJSaVFYUThJd0dybXE5c0RoVTlqTjA0SjdVL1RvRDFpNHE3VnlsRUc5DQorV1VGNkNLaEdBeTJIaEhwVncyTGFoOS9lUzdZMUZ1YURrWmhPZG1laG1BOCtqdHNZamJadnR5Mm1SWlF0UUZzDQpUU1VUUjREbUR2bVVPRVRmeStpRHdzK2RkWXVNTnJGeVVYV2dkMnpBQU4ydVl1UHFGY2pRcFNPODFzVTJTU3R3DQoxVzAyeUtYOGJEYmZFdjBzbUh3UzliQnFlSGo5NEM1Mjg0YXpsdTBmaUdpTm1OUEM4ckJLRmhBPQ0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQ==
        service:
          name: webhook-service
          namespace: metallb-system
          path: /convert
      conversionReviewVersions:
      - v1beta1
      - v1beta2
  group: metallb.io
  names:
    kind: BGPPeer
    listKind: BGPPeerList
    plural: bgppeers
    singular: bgppeer
  scope: Namespaced
  versions:
  - additionalPrinterColumns:
    - jsonPath: .spec.peerAddress
      name: Address
      type: string
    - jsonPath: .spec.peerASN
      name: ASN
      type: string
    - jsonPath: .spec.bfdProfile
      name: BFD Profile
      type: string
    - jsonPath: .spec.ebgpMultiHop
      name: Multi Hops
      type: string
    name: v1beta1
    schema:
      openAPIV3Schema:
        description: BGPPeer is the Schema for the peers API.
        properties:
          apiVersion:
            description: 'APIVersion defines the versioned schema of this representation
              of an object. Servers should convert recognized schemas to the latest
              internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
            type: string
          kind:
            description: 'Kind is a string value representing the REST resource this
              object represents. Servers may infer this from the endpoint the client
              submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
            type: string
          metadata:
            type: object
          spec:
            description: BGPPeerSpec defines the desired state of Peer.
            properties:
              bfdProfile:
                type: string
              ebgpMultiHop:
                description: EBGP peer is multi-hops away
                type: boolean
              holdTime:
                description: Requested BGP hold time, per RFC4271.
                type: string
              keepaliveTime:
                description: Requested BGP keepalive time, per RFC4271.
                type: string
              myASN:
                description: AS number to use for the local end of the session.
                format: int32
                maximum: 4294967295
                minimum: 0
                type: integer
              nodeSelectors:
                description: Only connect to this peer on nodes that match one of
                  these selectors.
                items:
                  properties:
                    matchExpressions:
                      items:
                        properties:
                          key:
                            type: string
                          operator:
                            type: string
                          values:
                            items:
                              type: string
                            minItems: 1
                            type: array
                        required:
                        - key
                        - operator
                        - values
                        type: object
                      type: array
                    matchLabels:
                      additionalProperties:
                        type: string
                      type: object
                  type: object
                type: array
              password:
                description: Authentication password for routers enforcing TCP MD5
                  authenticated sessions
                type: string
              peerASN:
                description: AS number to expect from the remote end of the session.
                format: int32
                maximum: 4294967295
                minimum: 0
                type: integer
              peerAddress:
                description: Address to dial when establishing the session.
                type: string
              peerPort:
                description: Port to dial when establishing the session.
                maximum: 16384
                minimum: 0
                type: integer
              routerID:
                description: BGP router ID to advertise to the peer
                type: string
              sourceAddress:
                description: Source address to use when establishing the session.
                type: string
            required:
            - myASN
            - peerASN
            - peerAddress
            type: object
          status:
            description: BGPPeerStatus defines the observed state of Peer.
            type: object
        type: object
    served: true
    storage: false
    subresources:
      status: {}
  - additionalPrinterColumns:
    - jsonPath: .spec.peerAddress
      name: Address
      type: string
    - jsonPath: .spec.peerASN
      name: ASN
      type: string
    - jsonPath: .spec.bfdProfile
      name: BFD Profile
      type: string
    - jsonPath: .spec.ebgpMultiHop
      name: Multi Hops
      type: string
    name: v1beta2
    schema:
      openAPIV3Schema:
        description: BGPPeer is the Schema for the peers API.
        properties:
          apiVersion:
            description: 'APIVersion defines the versioned schema of this representation
              of an object. Servers should convert recognized schemas to the latest
              internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
            type: string
          kind:
            description: 'Kind is a string value representing the REST resource this
              object represents. Servers may infer this from the endpoint the client
              submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
            type: string
          metadata:
            type: object
          spec:
            description: BGPPeerSpec defines the desired state of Peer.
            properties:
              bfdProfile:
                description: The name of the BFD Profile to be used for the BFD session
                  associated to the BGP session. If not set, the BFD session won't
                  be set up.
                type: string
              ebgpMultiHop:
                description: To set if the BGPPeer is multi-hops away. Needed for
                  FRR mode only.
                type: boolean
              holdTime:
                description: Requested BGP hold time, per RFC4271.
                type: string
              keepaliveTime:
                description: Requested BGP keepalive time, per RFC4271.
                type: string
              myASN:
                description: AS number to use for the local end of the session.
                format: int32
                maximum: 4294967295
                minimum: 0
                type: integer
              nodeSelectors:
                description: Only connect to this peer on nodes that match one of
                  these selectors.
                items:
                  description: A label selector is a label query over a set of resources.
                    The result of matchLabels and matchExpressions are ANDed. An empty
                    label selector matches all objects. A null label selector matches
                    no objects.
                  properties:
                    matchExpressions:
                      description: matchExpressions is a list of label selector requirements.
                        The requirements are ANDed.
                      items:
                        description: A label selector requirement is a selector that
                          contains values, a key, and an operator that relates the
                          key and values.
                        properties:
                          key:
                            description: key is the label key that the selector applies
                              to.
                            type: string
                          operator:
                            description: operator represents a key's relationship
                              to a set of values. Valid operators are In, NotIn, Exists
                              and DoesNotExist.
                            type: string
                          values:
                            description: values is an array of string values. If the
                              operator is In or NotIn, the values array must be non-empty.
                              If the operator is Exists or DoesNotExist, the values
                              array must be empty. This array is replaced during a
                              strategic merge patch.
                            items:
                              type: string
                            type: array
                        required:
                        - key
                        - operator
                        type: object
                      type: array
                    matchLabels:
                      additionalProperties:
                        type: string
                      description: matchLabels is a map of {key,value} pairs. A single
                        {key,value} in the matchLabels map is equivalent to an element
                        of matchExpressions, whose key field is "key", the operator
                        is "In", and the values array contains only "value". The requirements
                        are ANDed.
                      type: object
                  type: object
                type: array
              password:
                description: Authentication password for routers enforcing TCP MD5
                  authenticated sessions
                type: string
              passwordSecret:
                description: passwordSecret is name of the authentication secret for
                  BGP Peer. the secret must be of type "kubernetes.io/basic-auth",
                  and created in the same namespace as the MetalLB deployment. The
                  password is stored in the secret as the key "password".
                properties:
                  name:
                    description: name is unique within a namespace to reference a
                      secret resource.
                    type: string
                  namespace:
                    description: namespace defines the space within which the secret
                      name must be unique.
                    type: string
                type: object
              peerASN:
                description: AS number to expect from the remote end of the session.
                format: int32
                maximum: 4294967295
                minimum: 0
                type: integer
              peerAddress:
                description: Address to dial when establishing the session.
                type: string
              peerPort:
                default: 179
                description: Port to dial when establishing the session.
                maximum: 16384
                minimum: 0
                type: integer
              routerID:
                description: BGP router ID to advertise to the peer
                type: string
              sourceAddress:
                description: Source address to use when establishing the session.
                type: string
            required:
            - myASN
            - peerASN
            - peerAddress
            type: object
          status:
            description: BGPPeerStatus defines the observed state of Peer.
            type: object
        type: object
    served: true
    storage: true
    subresources:
      status: {}
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: []
  storedVersions: []
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  annotations:
    controller-gen.kubebuilder.io/version: v0.7.0
  creationTimestamp: null
  name: communities.metallb.io
spec:
  group: metallb.io
  names:
    kind: Community
    listKind: CommunityList
    plural: communities
    singular: community
  scope: Namespaced
  versions:
  - name: v1beta1
    schema:
      openAPIV3Schema:
        description: Community is a collection of aliases for communities. Users can
          define named aliases to be used in the BGPPeer CRD.
        properties:
          apiVersion:
            description: 'APIVersion defines the versioned schema of this representation
              of an object. Servers should convert recognized schemas to the latest
              internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
            type: string
          kind:
            description: 'Kind is a string value representing the REST resource this
              object represents. Servers may infer this from the endpoint the client
              submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
            type: string
          metadata:
            type: object
          spec:
            description: CommunitySpec defines the desired state of Community.
            properties:
              communities:
                items:
                  properties:
                    name:
                      description: The name of the alias for the community.
                      type: string
                    value:
                      description: The BGP community value corresponding to the given
                        name.
                      type: string
                  type: object
                type: array
            type: object
          status:
            description: CommunityStatus defines the observed state of Community.
            type: object
        type: object
    served: true
    storage: true
    subresources:
      status: {}
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: []
  storedVersions: []
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  annotations:
    controller-gen.kubebuilder.io/version: v0.7.0
  creationTimestamp: null
  name: ipaddresspools.metallb.io
spec:
  group: metallb.io
  names:
    kind: IPAddressPool
    listKind: IPAddressPoolList
    plural: ipaddresspools
    singular: ipaddresspool
  scope: Namespaced
  versions:
  - additionalPrinterColumns:
    - jsonPath: .spec.autoAssign
      name: Auto Assign
      type: boolean
    - jsonPath: .spec.avoidBuggyIPs
      name: Avoid Buggy IPs
      type: boolean
    - jsonPath: .spec.addresses
      name: Addresses
      type: string
    name: v1beta1
    schema:
      openAPIV3Schema:
        description: IPAddressPool represents a pool of IP addresses that can be allocated
          to LoadBalancer services.
        properties:
          apiVersion:
            description: 'APIVersion defines the versioned schema of this representation
              of an object. Servers should convert recognized schemas to the latest
              internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
            type: string
          kind:
            description: 'Kind is a string value representing the REST resource this
              object represents. Servers may infer this from the endpoint the client
              submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
            type: string
          metadata:
            type: object
          spec:
            description: IPAddressPoolSpec defines the desired state of IPAddressPool.
            properties:
              addresses:
                description: A list of IP address ranges over which MetalLB has authority.
                  You can list multiple ranges in a single pool, they will all share
                  the same settings. Each range can be either a CIDR prefix, or an
                  explicit start-end range of IPs.
                items:
                  type: string
                type: array
              autoAssign:
                default: true
                description: AutoAssign flag used to prevent MetallB from automatic
                  allocation for a pool.
                type: boolean
              avoidBuggyIPs:
                default: false
                description: AvoidBuggyIPs prevents addresses ending with .0 and .255
                  to be used by a pool.
                type: boolean
            required:
            - addresses
            type: object
          status:
            description: IPAddressPoolStatus defines the observed state of IPAddressPool.
            type: object
        required:
        - spec
        type: object
    served: true
    storage: true
    subresources:
      status: {}
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: []
  storedVersions: []
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  annotations:
    controller-gen.kubebuilder.io/version: v0.7.0
  creationTimestamp: null
  name: l2advertisements.metallb.io
spec:
  group: metallb.io
  names:
    kind: L2Advertisement
    listKind: L2AdvertisementList
    plural: l2advertisements
    singular: l2advertisement
  scope: Namespaced
  versions:
  - additionalPrinterColumns:
    - jsonPath: .spec.ipAddressPools
      name: IPAddressPools
      type: string
    - jsonPath: .spec.ipAddressPoolSelectors
      name: IPAddressPool Selectors
      type: string
    - jsonPath: .spec.interfaces
      name: Interfaces
      type: string
    - jsonPath: .spec.nodeSelectors
      name: Node Selectors
      priority: 10
      type: string
    name: v1beta1
    schema:
      openAPIV3Schema:
        description: L2Advertisement allows to advertise the LoadBalancer IPs provided
          by the selected pools via L2.
        properties:
          apiVersion:
            description: 'APIVersion defines the versioned schema of this representation
              of an object. Servers should convert recognized schemas to the latest
              internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
            type: string
          kind:
            description: 'Kind is a string value representing the REST resource this
              object represents. Servers may infer this from the endpoint the client
              submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
            type: string
          metadata:
            type: object
          spec:
            description: L2AdvertisementSpec defines the desired state of L2Advertisement.
            properties:
              interfaces:
                description: A list of interfaces to announce from. The LB IP will
                  be announced only from these interfaces. If the field is not set,
                  we advertise from all the interfaces on the host.
                items:
                  type: string
                type: array
              ipAddressPoolSelectors:
                description: A selector for the IPAddressPools which would get advertised
                  via this advertisement. If no IPAddressPool is selected by this
                  or by the list, the advertisement is applied to all the IPAddressPools.
                items:
                  description: A label selector is a label query over a set of resources.
                    The result of matchLabels and matchExpressions are ANDed. An empty
                    label selector matches all objects. A null label selector matches
                    no objects.
                  properties:
                    matchExpressions:
                      description: matchExpressions is a list of label selector requirements.
                        The requirements are ANDed.
                      items:
                        description: A label selector requirement is a selector that
                          contains values, a key, and an operator that relates the
                          key and values.
                        properties:
                          key:
                            description: key is the label key that the selector applies
                              to.
                            type: string
                          operator:
                            description: operator represents a key's relationship
                              to a set of values. Valid operators are In, NotIn, Exists
                              and DoesNotExist.
                            type: string
                          values:
                            description: values is an array of string values. If the
                              operator is In or NotIn, the values array must be non-empty.
                              If the operator is Exists or DoesNotExist, the values
                              array must be empty. This array is replaced during a
                              strategic merge patch.
                            items:
                              type: string
                            type: array
                        required:
                        - key
                        - operator
                        type: object
                      type: array
                    matchLabels:
                      additionalProperties:
                        type: string
                      description: matchLabels is a map of {key,value} pairs. A single
                        {key,value} in the matchLabels map is equivalent to an element
                        of matchExpressions, whose key field is "key", the operator
                        is "In", and the values array contains only "value". The requirements
                        are ANDed.
                      type: object
                  type: object
                type: array
              ipAddressPools:
                description: The list of IPAddressPools to advertise via this advertisement,
                  selected by name.
                items:
                  type: string
                type: array
              nodeSelectors:
                description: NodeSelectors allows to limit the nodes to announce as
                  next hops for the LoadBalancer IP. When empty, all the nodes having  are
                  announced as next hops.
                items:
                  description: A label selector is a label query over a set of resources.
                    The result of matchLabels and matchExpressions are ANDed. An empty
                    label selector matches all objects. A null label selector matches
                    no objects.
                  properties:
                    matchExpressions:
                      description: matchExpressions is a list of label selector requirements.
                        The requirements are ANDed.
                      items:
                        description: A label selector requirement is a selector that
                          contains values, a key, and an operator that relates the
                          key and values.
                        properties:
                          key:
                            description: key is the label key that the selector applies
                              to.
                            type: string
                          operator:
                            description: operator represents a key's relationship
                              to a set of values. Valid operators are In, NotIn, Exists
                              and DoesNotExist.
                            type: string
                          values:
                            description: values is an array of string values. If the
                              operator is In or NotIn, the values array must be non-empty.
                              If the operator is Exists or DoesNotExist, the values
                              array must be empty. This array is replaced during a
                              strategic merge patch.
                            items:
                              type: string
                            type: array
                        required:
                        - key
                        - operator
                        type: object
                      type: array
                    matchLabels:
                      additionalProperties:
                        type: string
                      description: matchLabels is a map of {key,value} pairs. A single
                        {key,value} in the matchLabels map is equivalent to an element
                        of matchExpressions, whose key field is "key", the operator
                        is "In", and the values array contains only "value". The requirements
                        are ANDed.
                      type: object
                  type: object
                type: array
            type: object
          status:
            description: L2AdvertisementStatus defines the observed state of L2Advertisement.
            type: object
        type: object
    served: true
    storage: true
    subresources:
      status: {}
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: []
  storedVersions: []
---
apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    app: metallb
  name: controller
  namespace: metallb-system
---
apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    app: metallb
  name: speaker
  namespace: metallb-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  labels:
    app: metallb
  name: controller
  namespace: metallb-system
rules:
- apiGroups:
  - ""
  resources:
  - secrets
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - ""
  resourceNames:
  - memberlist
  resources:
  - secrets
  verbs:
  - list
- apiGroups:
  - apps
  resourceNames:
  - controller
  resources:
  - deployments
  verbs:
  - get
- apiGroups:
  - metallb.io
  resources:
  - bgppeers
  verbs:
  - get
  - list
- apiGroups:
  - metallb.io
  resources:
  - addresspools
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - metallb.io
  resources:
  - bfdprofiles
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - metallb.io
  resources:
  - ipaddresspools
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - metallb.io
  resources:
  - bgpadvertisements
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - metallb.io
  resources:
  - l2advertisements
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - metallb.io
  resources:
  - communities
  verbs:
  - get
  - list
  - watch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  labels:
    app: metallb
  name: pod-lister
  namespace: metallb-system
rules:
- apiGroups:
  - ""
  resources:
  - pods
  verbs:
  - list
- apiGroups:
  - ""
  resources:
  - secrets
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - metallb.io
  resources:
  - addresspools
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - metallb.io
  resources:
  - bfdprofiles
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - metallb.io
  resources:
  - bgppeers
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - metallb.io
  resources:
  - l2advertisements
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - metallb.io
  resources:
  - bgpadvertisements
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - metallb.io
  resources:
  - ipaddresspools
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - metallb.io
  resources:
  - communities
  verbs:
  - get
  - list
  - watch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    app: metallb
  name: metallb-system:controller
rules:
- apiGroups:
  - ""
  resources:
  - services
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - ""
  resources:
  - services/status
  verbs:
  - update
- apiGroups:
  - ""
  resources:
  - events
  verbs:
  - create
  - patch
- apiGroups:
  - policy
  resourceNames:
  - controller
  resources:
  - podsecuritypolicies
  verbs:
  - use
- apiGroups:
  - admissionregistration.k8s.io
  resources:
  - validatingwebhookconfigurations
  - mutatingwebhookconfigurations
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - apiextensions.k8s.io
  resources:
  - customresourcedefinitions
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    app: metallb
  name: metallb-system:speaker
rules:
- apiGroups:
  - ""
  resources:
  - services
  - endpoints
  - nodes
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - discovery.k8s.io
  resources:
  - endpointslices
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - ""
  resources:
  - events
  verbs:
  - create
  - patch
- apiGroups:
  - policy
  resourceNames:
  - speaker
  resources:
  - podsecuritypolicies
  verbs:
  - use
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  labels:
    app: metallb
  name: controller
  namespace: metallb-system
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: controller
subjects:
- kind: ServiceAccount
  name: controller
  namespace: metallb-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  labels:
    app: metallb
  name: pod-lister
  namespace: metallb-system
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: pod-lister
subjects:
- kind: ServiceAccount
  name: speaker
  namespace: metallb-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  labels:
    app: metallb
  name: metallb-system:controller
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: metallb-system:controller
subjects:
- kind: ServiceAccount
  name: controller
  namespace: metallb-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  labels:
    app: metallb
  name: metallb-system:speaker
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: metallb-system:speaker
subjects:
- kind: ServiceAccount
  name: speaker
  namespace: metallb-system
---
{{- if eq .Mode "BGP"}}
apiVersion: v1
data:
  daemons: |
    # This file tells the frr package which daemons to start.
    #
    # Sample configurations for these daemons can be found in
    # /usr/share/doc/frr/examples/.
    #
    # ATTENTION:
    #
    # When activating a daemon for the first time, a config file, even if it is
    # empty, has to be present *and* be owned by the user and group "frr", else
    # the daemon will not be started by /etc/init.d/frr. The permissions should
    # be u=rw,g=r,o=.
    # When using "vtysh" such a config file is also needed. It should be owned by
    # group "frrvty" and set to ug=rw,o= though. Check /etc/pam.d/frr, too.
    #
    # The watchfrr and zebra daemons are always started.
    #
    bgpd=yes
    ospfd=no
    ospf6d=no
    ripd=no
    ripngd=no
    isisd=no
    pimd=no
    ldpd=no
    nhrpd=no
    eigrpd=no
    babeld=no
    sharpd=no
    pbrd=no
    bfdd=yes
    fabricd=no
    vrrpd=no

    #
    # If this option is set the /etc/init.d/frr script automatically loads
    # the config via "vtysh -b" when the servers are started.
    # Check /etc/pam.d/frr if you intend to use "vtysh"!
    #
    vtysh_enable=yes
    zebra_options="  -A 127.0.0.1 -s 90000000"
    bgpd_options="   -A 127.0.0.1 -p 0"
    ospfd_options="  -A 127.0.0.1"
    ospf6d_options=" -A ::1"
    ripd_options="   -A 127.0.0.1"
    ripngd_options=" -A ::1"
    isisd_options="  -A 127.0.0.1"
    pimd_options="   -A 127.0.0.1"
    ldpd_options="   -A 127.0.0.1"
    nhrpd_options="  -A 127.0.0.1"
    eigrpd_options=" -A 127.0.0.1"
    babeld_options=" -A 127.0.0.1"
    sharpd_options=" -A 127.0.0.1"
    pbrd_options="   -A 127.0.0.1"
    staticd_options="-A 127.0.0.1"
    bfdd_options="   -A 127.0.0.1"
    fabricd_options="-A 127.0.0.1"
    vrrpd_options="  -A 127.0.0.1"

    # configuration profile
    #
    #frr_profile="traditional"
    #frr_profile="datacenter"

    #
    # This is the maximum number of FD's that will be available.
    # Upon startup this is read by the control files and ulimit
    # is called. Uncomment and use a reasonable value for your
    # setup if you are expecting a large number of peers in
    # say BGP.
    #MAX_FDS=1024

    # The list of daemons to watch is automatically generated by the init script.
    #watchfrr_options=""

    # for debugging purposes, you can specify a "wrap" command to start instead
    # of starting the daemon directly, e.g. to use valgrind on ospfd:
    #   ospfd_wrap="/usr/bin/valgrind"
    # or you can use "all_wrap" for all daemons, e.g. to use perf record:
    #   all_wrap="/usr/bin/perf record --call-graph -"
    # the normal daemon command is added to this at the end.
  frr.conf: |
    ! This file gets overriden the first time the speaker renders a config.
    ! So anything configured here is only temporary.
    frr version 7.5.1
    frr defaults traditional
    hostname Router
    line vty
    log file /etc/frr/frr.log informational
  vtysh.conf: |
    service integrated-vtysh-config
kind: ConfigMap
metadata:
  name: frr-startup
  namespace: metallb-system
---
{{- end }}
apiVersion: v1
kind: Secret
metadata:
  name: webhook-server-cert
  namespace: metallb-system
---
apiVersion: v1
kind: Service
metadata:
  name: webhook-service
  namespace: metallb-system
spec:
  ports:
  - port: 443
    targetPort: 9443
  selector:
    component: controller
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: metallb
    component: controller
  name: controller
  namespace: metallb-system
spec:
  revisionHistoryLimit: 3
  selector:
    matchLabels:
      app: metallb
      component: controller
  template:
    metadata:
      annotations:
        prometheus.io/port: "7472"
        prometheus.io/scrape: "true"
      labels:
        app: metallb
        component: controller
    spec:
      tolerations:
        - key: node-role.kubernetes.io/master
          effect: NoSchedule
        - key: node-role.kubernetes.io/control-plane
          effect: NoSchedule
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: node-role.kubernetes.io/master
                operator: In
                values:
                - ""
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: node-role.kubernetes.io/control-plane
                operator: In
                values:
                - ""
      containers:
      - args:
        - --port=7472
        - --log-level=info
        env:
        {{- if eq .Mode "BGP"}}
        - name: METALLB_BGP_TYPE
          value: frr
        {{- end }}
        - name: METALLB_ML_SECRET_NAME
          value: memberlist
        - name: METALLB_DEPLOYMENT
          value: controller
        image: {{with .ImageRepoMirror}}{{.}}/{{end}}caas4/metallb-controller:{{.Version}}
        livenessProbe:
          failureThreshold: 3
          httpGet:
            path: /metrics
            port: monitoring
          initialDelaySeconds: 10
          periodSeconds: 10
          successThreshold: 1
          timeoutSeconds: 1
        name: controller
        ports:
        - containerPort: 7472
          name: monitoring
        - containerPort: 9443
          name: webhook-server
          protocol: TCP
        readinessProbe:
          failureThreshold: 3
          httpGet:
            path: /metrics
            port: monitoring
          initialDelaySeconds: 10
          periodSeconds: 10
          successThreshold: 1
          timeoutSeconds: 1
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
            - all
          readOnlyRootFilesystem: true
        volumeMounts:
        - mountPath: /tmp/k8s-webhook-server/serving-certs
          name: cert
          readOnly: true
      nodeSelector:
        kubernetes.io/os: linux
      securityContext:
        fsGroup: 65534
        runAsNonRoot: true
        runAsUser: 65534
      serviceAccountName: controller
      terminationGracePeriodSeconds: 0
      volumes:
      - name: cert
        secret:
          defaultMode: 420
          secretName: webhook-server-cert
---
apiVersion: apps/v1
kind: DaemonSet
metadata:
  labels:
    app: metallb
    component: speaker
  name: speaker
  namespace: metallb-system
spec:
  selector:
    matchLabels:
      app: metallb
      component: speaker
  template:
    metadata:
      annotations:
        prometheus.io/port: "7472"
        prometheus.io/scrape: "true"
      labels:
        app: metallb
        component: speaker
    spec:
      tolerations:
        - key: node-role.kubernetes.io/master
          effect: NoSchedule
        - key: node-role.kubernetes.io/control-plane
          effect: NoSchedule
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: node-role.kubernetes.io/master
                operator: In
                values:
                - ""
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: node-role.kubernetes.io/control-plane
                operator: In
                values:
                - ""
      containers:
      {{- if eq .Mode "BGP"}}
      - command:
        - /bin/sh
        - -c
        - |
          /sbin/tini -- /usr/lib/frr/docker-start &
          attempts=0
          until [[ -f /etc/frr/frr.log || $attempts -eq 60 ]]; do
            sleep 1
            attempts=$(( $attempts + 1 ))
          done
          tail -f /etc/frr/frr.log
        env:
        - name: TINI_SUBREAPER
          value: "true"
        image: {{with .ImageRepoMirror}}{{.}}/{{end}}caas4/frr:v7.5.1
        name: frr
        securityContext:
          capabilities:
            add:
            - NET_ADMIN
            - NET_RAW
            - SYS_ADMIN
            - NET_BIND_SERVICE
        volumeMounts:
        - mountPath: /var/run/frr
          name: frr-sockets
        - mountPath: /etc/frr
          name: frr-conf
      - command:
        - /etc/frr_reloader/frr-reloader.sh
        image: {{with .ImageRepoMirror}}{{.}}/{{end}}caas4/frr:v7.5.1
        name: reloader
        volumeMounts:
        - mountPath: /var/run/frr
          name: frr-sockets
        - mountPath: /etc/frr
          name: frr-conf
        - mountPath: /etc/frr_reloader
          name: reloader
      - args:
        - --metrics-port=7473
        command:
        - /etc/frr_metrics/frr-metrics
        image: {{with .ImageRepoMirror}}{{.}}/{{end}}caas4/frr:v7.5.1
        name: frr-metrics
        ports:
        - containerPort: 7473
          name: monitoring
        volumeMounts:
        - mountPath: /var/run/frr
          name: frr-sockets
        - mountPath: /etc/frr
          name: frr-conf
        - mountPath: /etc/frr_metrics
          name: metrics
      {{- end }}
      - args:
        - --port=7472
        - --log-level=info
        env:
        {{- if eq .Mode "BGP"}}
        - name: FRR_CONFIG_FILE
          value: /etc/frr_reloader/frr.conf
        - name: FRR_RELOADER_PID_FILE
          value: /etc/frr_reloader/reloader.pid
        - name: METALLB_BGP_TYPE
          value: frr
        {{- end }}
        - name: METALLB_NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        - name: METALLB_HOST
          valueFrom:
            fieldRef:
              fieldPath: status.hostIP
        - name: METALLB_ML_BIND_ADDR
          valueFrom:
            fieldRef:
              fieldPath: status.podIP
        - name: METALLB_ML_LABELS
          value: app=metallb,component=speaker
        - name: METALLB_ML_SECRET_KEY
          valueFrom:
            secretKeyRef:
              key: secretkey
              name: memberlist
        image: {{with .ImageRepoMirror}}{{.}}/{{end}}caas4/metallb-speaker:{{.Version}}
        livenessProbe:
          failureThreshold: 3
          httpGet:
            path: /metrics
            port: monitoring
          initialDelaySeconds: 10
          periodSeconds: 10
          successThreshold: 1
          timeoutSeconds: 1
        name: speaker
        ports:
        - containerPort: 7472
          name: monitoring
        - containerPort: 7946
          name: memberlist-tcp
        - containerPort: 7946
          name: memberlist-udp
          protocol: UDP
        readinessProbe:
          failureThreshold: 3
          httpGet:
            path: /metrics
            port: monitoring
          initialDelaySeconds: 10
          periodSeconds: 10
          successThreshold: 1
          timeoutSeconds: 1
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            add:
            - NET_RAW
            drop:
            - ALL
          readOnlyRootFilesystem: true
        {{- if eq .Mode "BGP"}}
        volumeMounts:
        - mountPath: /etc/frr_reloader
          name: reloader
        {{- end }}
      hostNetwork: true
      {{- if eq .Mode "BGP"}}
      initContainers:
      - command:
        - /bin/sh
        - -c
        - cp -rLf /tmp/frr/* /etc/frr/
        image: {{with .ImageRepoMirror}}{{.}}/{{end}}caas4/frr:v7.5.1
        name: cp-frr-files
        securityContext:
          runAsGroup: 101
          runAsUser: 100
        volumeMounts:
        - mountPath: /tmp/frr
          name: frr-startup
        - mountPath: /etc/frr
          name: frr-conf
      - command:
        - /bin/sh
        - -c
        - cp -f /frr-reloader.sh /etc/frr_reloader/
        image: {{with .ImageRepoMirror}}{{.}}/{{end}}caas4/metallb-speaker:{{.Version}}
        name: cp-reloader
        volumeMounts:
        - mountPath: /etc/frr_reloader
          name: reloader
      - command:
        - /bin/sh
        - -c
        - cp -f /frr-metrics /etc/frr_metrics/
        image: {{with .ImageRepoMirror}}{{.}}/{{end}}caas4/metallb-speaker:{{.Version}}
        name: cp-metrics
        volumeMounts:
        - mountPath: /etc/frr_metrics
          name: metrics
      {{- end }}
      nodeSelector:
        kubernetes.io/os: linux
      serviceAccountName: speaker
      {{- if eq .Mode "BGP"}}
      shareProcessNamespace: true
      {{- end }}
      terminationGracePeriodSeconds: 2
      tolerations:
      - effect: NoSchedule
        key: node-role.kubernetes.io/master
        operator: Exists
      - effect: NoSchedule
        key: node-role.kubernetes.io/control-plane
        operator: Exists
      {{- if eq .Mode "BGP"}}
      volumes:
      - emptyDir: {}
        name: frr-sockets
      - configMap:
          name: frr-startup
        name: frr-startup
      - emptyDir: {}
        name: frr-conf
      - emptyDir: {}
        name: reloader
      - emptyDir: {}
        name: metrics
      {{- end }}
---
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  creationTimestamp: null
  name: metallb-webhook-configuration
webhooks:
- admissionReviewVersions:
  - v1
  clientConfig:
    service:
      name: webhook-service
      namespace: metallb-system
      path: /validate-metallb-io-v1beta2-bgppeer
  failurePolicy: Fail
  name: bgppeersvalidationwebhook.metallb.io
  rules:
  - apiGroups:
    - metallb.io
    apiVersions:
    - v1beta2
    operations:
    - CREATE
    - UPDATE
    resources:
    - bgppeers
  sideEffects: None
- admissionReviewVersions:
  - v1
  clientConfig:
    service:
      name: webhook-service
      namespace: metallb-system
      path: /validate-metallb-io-v1beta1-addresspool
  failurePolicy: Fail
  name: addresspoolvalidationwebhook.metallb.io
  rules:
  - apiGroups:
    - metallb.io
    apiVersions:
    - v1beta1
    operations:
    - CREATE
    - UPDATE
    resources:
    - addresspools
  sideEffects: None
- admissionReviewVersions:
  - v1
  clientConfig:
    service:
      name: webhook-service
      namespace: metallb-system
      path: /validate-metallb-io-v1beta1-bfdprofile
  failurePolicy: Fail
  name: bfdprofilevalidationwebhook.metallb.io
  rules:
  - apiGroups:
    - metallb.io
    apiVersions:
    - v1beta1
    operations:
    - CREATE
    - DELETE
    resources:
    - bfdprofiles
  sideEffects: None
- admissionReviewVersions:
  - v1
  clientConfig:
    service:
      name: webhook-service
      namespace: metallb-system
      path: /validate-metallb-io-v1beta1-bgpadvertisement
  failurePolicy: Fail
  name: bgpadvertisementvalidationwebhook.metallb.io
  rules:
  - apiGroups:
    - metallb.io
    apiVersions:
    - v1beta1
    operations:
    - CREATE
    - UPDATE
    resources:
    - bgpadvertisements
  sideEffects: None
- admissionReviewVersions:
  - v1
  clientConfig:
    service:
      name: webhook-service
      namespace: metallb-system
      path: /validate-metallb-io-v1beta1-community
  failurePolicy: Fail
  name: communityvalidationwebhook.metallb.io
  rules:
  - apiGroups:
    - metallb.io
    apiVersions:
    - v1beta1
    operations:
    - CREATE
    - UPDATE
    resources:
    - communities
  sideEffects: None
- admissionReviewVersions:
  - v1
  clientConfig:
    service:
      name: webhook-service
      namespace: metallb-system
      path: /validate-metallb-io-v1beta1-ipaddresspool
  failurePolicy: Fail
  name: ipaddresspoolvalidationwebhook.metallb.io
  rules:
  - apiGroups:
    - metallb.io
    apiVersions:
    - v1beta1
    operations:
    - CREATE
    - UPDATE
    resources:
    - ipaddresspools
  sideEffects: None
- admissionReviewVersions:
  - v1
  clientConfig:
    service:
      name: webhook-service
      namespace: metallb-system
      path: /validate-metallb-io-v1beta1-l2advertisement
  failurePolicy: Fail
  name: l2advertisementvalidationwebhook.metallb.io
  rules:
  - apiGroups:
    - metallb.io
    apiVersions:
    - v1beta1
    operations:
    - CREATE
    - UPDATE
    resources:
    - l2advertisements
  sideEffects: None
`
