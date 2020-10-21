# Template-operator

> Template-operator for HyperCloud Service

**Architecture**

## prerequisite Install
- kubernetes

## Install Template Operator

- [CRD](#crd)
- [Namespace](#namespace)
- [ServiceAccount](#serviceaccount)
- [Role](#role)
- [RoleBinding](#rolebinding)
- [Deployment](#deployment)
- [Test](#test)
- [Changes](#changes)

---

#### crd
> Apply crd
- kubectl apply -f tmax.io_templates_crd.yaml ([파일](./deploy/crds/tmax.io_templates_crd.yaml))
- kubectl apply -f tmax.io_clustertemplates_crd.yaml ([파일](./deploy/crds/tmax.io_clustertemplates_crd.yaml))
- kubectl apply -f tmax.io_templateinstances_crd.yaml ([파일](./deploy/crds/tmax.io_templateinstances_crd.yaml))
- kubectl apply -f tmax.io_catalogserviceclaims_crd.yaml ([파일](./deploy/crds/tmax.io_catalogserviceclaims_crd.yaml))

---

#### Namespace
> 오퍼레이터를 위한 네임스페이스를 생성 합니다.
- kubectl create namespace {YOUR_NAMESPACE}

---

#### ServiceAccount
> 서비스어카운트를 생성 합니다.

- kubectl apply -f service_account.yaml -n {YOUR_NAMESPACE} ([파일](./deploy/service_account.yaml))

---

#### Role
> 서비스어카운트를 위한 Role을 생성 합니다.

- kubectl apply -f role.yaml -n {YOUR_NAMESPACE} ([파일](./deploy/role.yaml))

---

#### RoleBinding
> RoleBinding을 생성 합니다.
>> 단, role_binding.yaml 내부의 namespace를 {YOUR_NAMESPACE}로 변경해주어야 합니다.

- kubectl apply -f role_binding.yaml -n {YOUR_NAMESPACE} ([파일](./deploy/role_binding.yaml))

---

#### Deployment
> Template Operator를 생성 합니다.

- kubectl apply -f operator.yaml -n {YOUR_NAMESPACE} ([파일](./deploy/operator.yaml))

---

#### Test
> 테스트는 example directory를 참고하여 하시면 됩니다.

- kubectl apply -f example.yaml

---

#### Changes
> hypercloud 4.1 대비 변경 사항 입니다.
1. ClusterTemplate 추가
    - default 네임스페이스에 만들어서 사용 하던 사용자 공통 template을 cluster-scoped의 ClusterTemplate을 통해 사용
2. CatalogServiceClaim 변경
    - name-scope에서 cluster-scope로 변경
    - Approve 상태 추가 (Approve 후, 성공적으로 ClusterTemplate 만들어지면 Success 상태로 변경)
    - 승인 후, Template이 아닌 ClusterTemplate이 생성
3. TemplateInstance 사용법 변경
    - ClusterTemplate을 기반으로 instance 생성 할 경우 [파일](./example/cluster-example-template-instance.yaml)과 같이 사용
    - Template 기반으로 instance 생성 할 시에는 기존과 동일하게 사용 가능
4. template-operator 와 template-service-broker가 독립적으로 동작 할 수 있도록 로직 분리 