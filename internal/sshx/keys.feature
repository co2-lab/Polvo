# language: en
# @anchors
#   code: KYFTR
#   ref: KYSEX
#   updated_at: 2026-10-18
#   layer: feature

@KYSEX
Feature: Keys

  @KYSEX-B01 @unit-level
  Scenario: A identidade nasce no primeiro pedido e é a mesma no segundo
    Given uma pasta vazia e o caminho "id/polvo_ed25519" dentro dela
    When a identidade é pedida duas vezes nesse caminho
    Then o primeiro pedido diz que a identidade nasceu, o segundo diz que não, e as duas chaves públicas são iguais e do tipo "ssh-ed25519"

  @KYSEX-B02 @unit-level
  Scenario: A chave privada nasce só para o dono
    Given uma pasta vazia e o caminho "id/polvo_ed25519" dentro dela
    When a identidade é pedida nesse caminho
    Then o ficheiro tem as permissões 0600 e a pasta "id" tem as permissões 0700

  @KYSEX-B03 @unit-level
  Scenario: A chave da lista entra, a outra não, e a lista vale na verificação seguinte
    Given uma lista com um comentário, uma linha vazia e a chave A, e uma chave B fora da lista
    When as chaves A e B são verificadas, a chave B é acrescentada à lista e verificada outra vez
    Then a chave A entra, a chave B é recusada na primeira verificação e entra na segunda

  @KYSEX-B04 @unit-level
  Scenario: Uma linha que não é chave é saltada e comunicada
    Given uma lista com a linha "ssh-ed25519 lixo" na linha 1 e a chave A na linha 2
    When a chave A é verificada
    Then a chave A entra, e é comunicado o aviso da linha 1

  @KYSEX-B05 @unit-level
  Scenario: O primeiro contacto com o nó "10.0.0.1:2222" regista a chave dele
    Given uma lista de nós conhecidos vazia e a chave N do nó "10.0.0.1:2222"
    When o desktop se liga ao nó "10.0.0.1:2222" com a chave N
    Then a ligação é aceite e a lista passa a ter uma linha com a chave N para esse nó

  @KYSEX-B06 @unit-level
  Scenario: A segunda ligação com a mesma chave é aceite sem mudar a lista
    Given uma lista de nós conhecidos com a chave N do nó "10.0.0.1:2222"
    When o desktop se liga ao nó "10.0.0.1:2222" com a chave N
    Then a ligação é aceite e o conteúdo da lista é o mesmo de antes

  @KYSEX-B07 @unit-level
  Scenario: Uma chave diferente para um nó conhecido é recusada
    Given uma lista de nós conhecidos com a chave N do nó "10.0.0.1:2222"
    When o desktop se liga ao nó "10.0.0.1:2222" com a chave M
    Then a ligação é recusada com o erro de chave mudada para "10.0.0.1:2222" e o conteúdo da lista é o mesmo de antes

  @KYSEX-E01 @unit-level
  Scenario: Uma chave privada legível por outros é recusada
    Given uma identidade já criada cujo ficheiro passou a ter as permissões 0644
    When a identidade é pedida nesse caminho
    Then o pedido falha com o erro de permissões, com o caminho e as permissões 0644

  @KYSEX-E02 @unit-level
  Scenario: Um ficheiro que não é chave dá erro com o caminho
    Given o ficheiro "id/polvo_ed25519" com o texto "isto nao e uma chave" e as permissões 0600
    When a identidade é pedida nesse caminho
    Then o pedido falha com um erro que contém o caminho do ficheiro

  @KYSEX-E03 @unit-level
  Scenario: Sem a lista de chaves autorizadas ninguém entra
    Given um caminho de lista de chaves autorizadas que não existe
    When a chave A é verificada
    Then a chave A é recusada, sem erro

  @KYSEX-I01 @unit-level
  Scenario: A linha escrita para uma chave é aceite para essa chave
    Given a chave A e uma lista vazia
    When a linha de chave autorizada da chave A é escrita na lista
    Then a chave A entra

  @KYSEX-I02 @unit-level
  Scenario: A chave registada é aceite por uma lista nova lida do mesmo ficheiro
    Given uma lista de nós conhecidos vazia que registou a chave N do nó "10.0.0.1:2222" no primeiro contacto
    When outra lista é aberta sobre o mesmo ficheiro e o desktop se liga ao nó com a chave N
    Then a ligação é aceite

  @KYSEX-B08 @unit-level
  Scenario: A chave em bruto
    Given uma pasta sem identidade
    When se pede a chave em bruto
    Then a identidade nasce, e a chave pública da chave em bruto é a da identidade
