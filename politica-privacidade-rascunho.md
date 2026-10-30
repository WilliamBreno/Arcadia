# Política de Privacidade do Evve — RASCUNHO

> **Rascunho para revisão jurídica. Não publicar.** Escrito a partir do que o sistema faz hoje (outubro de 2026), para o advogado ajustar o texto, as bases legais e os dados da empresa. Os trechos entre colchetes precisam ser preenchidos. Itens marcados com ⚖️ pedem decisão ou validação jurídica.

## 1. Quem somos e como falar conosco

O Evve é uma plataforma de venda de ingressos e gestão de eventos operada por **[razão social]**, CNPJ **[número]**, com sede em **[endereço]** ("Evve", "nós").

Encarregado pelo tratamento de dados pessoais (DPO): **[nome]**, pelo e-mail **[e-mail do encarregado]**. ⚖️ Designar o encarregado (LGPD art. 41; Resolução CD/ANPD nº 18/2024).

## 2. Que dados tratamos, para quê e com qual base legal

| Situação | Dados | Para quê | Base legal (LGPD) ⚖️ |
|---|---|---|---|
| Criar conta e entrar | nome, e-mail, senha (guardada só como hash), ou login com Google (identificador e foto do perfil); código de acesso por e-mail (guardado só como hash, vale 10 min) | identificar você e proteger a conta | execução de contrato (art. 7º, V) |
| Comprar ou se inscrever | dados do pedido, ingressos, titular (nome e e-mail), valores, forma de pagamento | emitir e entregar o ingresso, atender cancelamentos, reembolsos e a revenda oficial | execução de contrato (art. 7º, V) |
| Pagar com cartão | número do cartão e CPF do titular são digitados no formulário do Mercado Pago e **não passam pelo Evve**; guardamos o identificador e o status do pagamento | processar o pagamento e prevenir fraude | execução de contrato e prevenção à fraude (art. 7º, V, e art. 11, II, "g" quando aplicável) |
| Check-in na entrada | código e QR do ingresso, hora da leitura, quem leu | validar a entrada no evento | execução de contrato (art. 7º, V) |
| Organizadores | nome público, CPF ou CNPJ, chave Pix, dados do evento | verificar o organizador e pagar os repasses | execução de contrato e obrigação legal/regulatória (art. 7º, II e V) |
| Participantes de concursos (cosplay, dança, canto) | nome, nome artístico, contato, data de nascimento, nome do responsável (para menores), materiais enviados | inscrição, avaliação pelos jurados e divulgação do resultado | execução de contrato; para crianças e adolescentes, consentimento do responsável e melhor interesse (art. 14) ⚖️ |
| Avaliar um evento | nota, comentário e nome curto ("Ana S.") | mostrar avaliações a outras pessoas | legítimo interesse (art. 7º, IX) ⚖️ |
| Seguir organizador ou categoria; lembretes | o que você segue, preferência de receber novidades | avisar de eventos novos, lembrar do seu evento, convite para avaliar | consentimento ao seguir (art. 7º, I) e execução de contrato nos lembretes do ingresso ⚖️ |
| Lembrete de compra não concluída | e-mail e o que estava no carrinho | lembrar uma vez que os ingressos ainda estão disponíveis; dá para recusar em todo e-mail | legítimo interesse (art. 7º, IX), com saída fácil ⚖️ |
| Lista de espera | ingresso e quantidade desejados, ordem na fila | avisar quando abrir vaga e guardá-la para você | execução de contrato (art. 7º, V) |
| Segurança | endereço IP (temporário, para limitar tentativas de login e de acesso), registros técnicos de erro | proteger contra abuso e fraude | legítimo interesse (art. 7º, IX) |
| Pixels de anúncio dos organizadores | ver a seção 4 | medir anúncios do organizador | consentimento (art. 7º, I) |

Não vendemos dados pessoais.

## 3. Com quem compartilhamos

- **Organizador do evento que você comprou:** nome e e-mail do titular, ingresso e situação do check-in, para a gestão do evento e a portaria. O organizador é controlador desses dados para as finalidades dele. ⚖️ Definir o papel (controlador conjunto ou independente) e o contrato com o organizador.
- **Mercado Pago:** processamento de pagamentos, estornos e contestações.
- **Prestadores que operam o serviço para nós (operadores):** hospedagem e banco de dados (Railway, Neon, Vercel), envio de e-mails (Resend), armazenamento de arquivos (**[provedor S3/R2]**). ⚖️ Contratos com cláusulas da LGPD (art. 39).
- **Autoridades**, quando a lei ou ordem judicial exigir.

**Transferência internacional:** parte desses prestadores guarda dados fora do Brasil (por exemplo, nos Estados Unidos). ⚖️ Fundamentar conforme os arts. 33 a 36 e a Resolução CD/ANPD nº 19/2024 (cláusulas-padrão).

## 4. Cookies e pixels

- **Sempre usados:** um cookie de sessão para manter você conectado com segurança e o armazenamento do navegador para lembrar escolhas. São necessários ao funcionamento.
- **Opcionais:** pixels de anúncio (Meta, Google Analytics 4, TikTok) que o organizador pode ligar. Só carregam se você aceitar no aviso de cookies; funcionam só nas páginas daquele evento e no seu pedido; medem visualização, início e conclusão da compra (valor e quantidade). Nome, e-mail, CPF e dados de pagamento não são enviados. Você pode mudar a escolha a qualquer momento na página **Cookies** (rodapé do site).

## 5. Por quanto tempo guardamos ⚖️

- Conta: enquanto estiver ativa.
- Pedidos, pagamentos, reembolsos e lançamentos financeiros: **[prazo legal fiscal/contábil, ex.: 5 anos]**.
- Código de acesso por e-mail: 10 minutos (depois só o registro de uso).
- Registros de controle de tentativas por IP: minutos a horas.
- Avaliações e fichas de concurso: **[definir]**.

## 6. Seus direitos

Você pode pedir confirmação de que tratamos seus dados, acesso, correção, anonimização, bloqueio ou eliminação de dados desnecessários, portabilidade, informação sobre compartilhamentos, e revogar consentimentos (art. 18). Peça pelo e-mail **[e-mail do encarregado]**; respondemos em até 15 dias. ⚖️ Ainda não há tela de "baixar meus dados" e "excluir minha conta" no site; hoje o atendimento seria manual.

Para deixar de receber avisos e lembretes, use o link "não quero receber" no próprio e-mail. Para pixels, use a página Cookies.

## 7. Segurança

Senhas e códigos de acesso guardados só como hash; conexão sempre criptografada (HTTPS); dados de cartão nunca passam pelo Evve; acesso interno restrito; limite de tentativas contra ataques. Em caso de incidente que possa gerar risco relevante, avisamos a ANPD e as pessoas afetadas no prazo legal (Resolução CD/ANPD nº 15/2024).

## 8. Crianças e adolescentes ⚖️

A compra exige conta de pessoa maior de idade **[validar]**. Em concursos com participantes menores de 18 anos, a inscrição é feita com o responsável legal, e só tratamos o necessário para a participação. ⚖️ Avaliar as obrigações do ECA Digital (Lei nº 15.211/2025).

## 9. Mudanças nesta política

Avisaremos mudanças relevantes no site e, quando necessário, por e-mail. Versão **[número]**, de **[data]**.
