# language: en
# @anchors
#   code: DSFST
#   ref: DSCVR
#   updated_at: 2026-10-18
#   layer: feature

@DSCVR
Feature: Discovery

  @DSCVR-B01 @unit-level
  Scenario: Anunciar
    Given um nó anunciado como "oficina" na máquina "oficina", porta 2222, com a impressão digital "SHA256:abc" e o endereço 192.0.2.7
    When chega uma pergunta pelo _polvo._tcp.local
    Then a resposta tem o PTR, o SRV com a porta 2222, o TXT com fp=SHA256:abc e name=oficina, e o A 192.0.2.7

  @DSCVR-B02 @unit-level
  Scenario: Procurar
    Given dois nós anunciados no grupo de teste, "a" e "b"
    When a janela procura durante meio segundo
    Then encontra os dois, cada um uma vez, com o endereço e a impressão digital

  @DSCVR-B03 @unit-level
  Scenario: A quem se responde
    Given um nó anunciado no grupo de teste
    When pergunta alguém da porta do grupo, e alguém de outra porta
    Then o primeiro recebe a resposta no grupo, e o segundo no endereço dele

  @DSCVR-B04 @unit-level
  Scenario: O endereço de um nó sem A
    Given um nó anunciado sem endereços
    When a janela o encontra
    Then o endereço é o de onde veio a resposta, com a porta do SRV

  @DSCVR-E01 @unit-level
  Scenario: Um grupo que não se abre
    Given um grupo que não é de multicast
    When um nó se anuncia nele
    Then dá o erro

  @DSCVR-E02 @unit-level
  Scenario: Um pacote que não é DNS
    Given um nó anunciado no grupo de teste
    When chega um pacote de lixo, e depois uma pergunta
    Then a pergunta tem resposta
