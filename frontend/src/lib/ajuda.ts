// Item 9.9: central de ajuda. As respostas seguem as regras que o sistema
// já cumpre (plano-eventos.md); ao mudar uma regra, mude o texto aqui.
// k: palavras extras só para a busca.
export type Pergunta = { p: string; r: string; k?: string }

export const faqComprador: Pergunta[] = [
  {
    p: 'Onde está o meu ingresso?',
    k: 'qr code baixar',
    r: 'Em "Meus ingressos", com o QR Code de cada pessoa. Ele também chega por e-mail logo depois do pagamento. Se o organizador usa QR dinâmico, abra o ingresso pelo site na hora de entrar: print não vale.',
  },
  {
    p: 'Quanto custa a taxa?',
    k: 'preço tarifa conveniência',
    r: 'A taxa é fixa por ingresso e depende da faixa de preço (veja em "Quanto custa"). Ela aparece antes de você pagar. Em alguns eventos o organizador paga a taxa por você. Ingresso gratuito não tem taxa.',
  },
  {
    p: 'Posso parcelar no cartão?',
    r: 'Sim, em até 12 vezes, com parcela mínima de R$ 20. Os juros do parcelamento são do cartão e aparecem no formulário antes de confirmar. No Pix o pagamento é à vista e a confirmação sai em segundos.',
  },
  {
    p: 'Desisti de ir. Recebo o dinheiro de volta?',
    k: 'reembolso estorno cancelar devolução arrependimento',
    r: 'Com a garantia de vaga, você cancela quando quiser até o início do evento e recebe tudo de volta, inclusive a taxa. Sem ela, vale o direito de arrependimento: até 7 dias depois da compra, desde que faltem mais de 48 horas para o evento.',
  },
  {
    p: 'E se o evento for cancelado?',
    k: 'reembolso estorno devolução adiado',
    r: 'Você recebe tudo de volta automaticamente, no mesmo meio de pagamento (Pix ou cartão), com taxa e garantia. Não precisa pedir.',
  },
  {
    p: 'Como passo o ingresso para outra pessoa?',
    k: 'transferir transferência trocar titular nome',
    r: 'Em "Meus ingressos", use "Transferir" e informe o nome e o e-mail de quem vai. O ingresso ganha um QR novo, enviado para a pessoa, e o QR antigo para de valer. Não tem taxa. Dá para transferir até o início do evento.',
  },
  {
    p: 'Não vou mais. Posso revender?',
    k: 'revenda vender repassar',
    r: 'Quando o ingresso esgota, você pode anunciá-lo na revenda oficial, em "Meus ingressos". Quem compra paga o preço de tabela e você recebe de volta o que pagou pelo ingresso quando a venda acontece. Fora do Evve, ninguém consegue garantir que o ingresso é verdadeiro.',
  },
  {
    p: 'Comprei de alguém nas redes sociais. É verdadeiro?',
    k: 'golpe falso fraude verificar',
    r: 'Digite o código em "Verificar ingresso". Mesmo um ingresso válido pode ter sido vendido para várias pessoas por print: a única troca segura é a transferência pelo Evve (o QR antigo deixa de valer na hora) ou a revenda oficial.',
  },
  {
    p: 'Como funciona a meia-entrada?',
    r: 'Compre o ingresso marcado como meia-entrada e leve o documento que dá direito (estudante, pessoa com deficiência, jovem de baixa renda). A conferência é na entrada.',
  },
  {
    p: 'Não recebi o e-mail.',
    r: 'Confira a caixa de spam e o endereço da sua conta. O ingresso está sempre em "Meus ingressos", mesmo sem o e-mail.',
  },
]

export const faqOrganizador: Pergunta[] = [
  {
    p: 'Quanto custa vender no Evve?',
    r: 'Criar e publicar evento é grátis. Por ingresso vendido há uma taxa fixa por faixa de preço, que o comprador paga ou que você escolhe pagar. Em evento gratuito não há taxa. A taxa do processador de pagamento sai do seu repasse.',
  },
  {
    p: 'Quando recebo o dinheiro das vendas?',
    k: 'repasse pix saque pagamento prazo',
    r: 'Depois do fim do evento, por Pix na chave do seu perfil: 1 dia útil para organizador verificado, 3 dias úteis para os demais. Em vendas no cartão, uma parte pode ficar retida por um tempo como reserva contra contestações.',
  },
  {
    p: 'Posso vender na porta no dia do evento?',
    r: 'Sim. Pelo celular, em "Vender na porta" (no check-in), em Pix ou dinheiro. No dinheiro, a taxa da plataforma é descontada do seu repasse.',
  },
  {
    p: 'Como funciona o check-in?',
    r: 'Abra o check-in no celular e leia o QR de cada pessoa. Funciona sem internet se você baixar o pacote do evento antes. A equipe com função de check-in também pode ler. Ingressos vendidos em outra plataforma podem ser importados por CSV.',
  },
  {
    p: 'Como dou acesso à minha equipe?',
    r: 'Em "Equipe", adicione a pessoa pelo e-mail e escolha a função: gestor, financeiro, leitura ou check-in. Pix, equipe, coprodução e cancelamento de evento ficam só com você.',
  },
  {
    p: 'Faço o evento com outro produtor. Dá para dividir o repasse?',
    r: 'Sim, pela coprodução: no evento, convide o outro organizador com um percentual. Cada um recebe a sua parte no próprio Pix. Depois da primeira venda, mudanças precisam do aceite dele.',
  },
  {
    p: 'Preciso cancelar o evento. O que acontece?',
    k: 'reembolso estorno devolução',
    r: 'Todos os compradores recebem o dinheiro de volta automaticamente, com taxa e garantia. Avise o público pela página do evento e pelo e-mail para quem segue você.',
  },
  {
    p: 'Como funciona a garantia de vaga para o meu público?',
    r: 'É opcional para o comprador e não custa nada para você: ele paga um valor a mais e pode cancelar até o início do evento com tudo de volta. O valor cancelado não entra no seu repasse.',
  },
]

export const assuntosSuporte: { valor: string; rotulo: string }[] = [
  { valor: 'compra', rotulo: 'Compra e pagamento' },
  { valor: 'ingresso', rotulo: 'Ingresso, QR ou transferência' },
  { valor: 'reembolso', rotulo: 'Cancelamento e reembolso' },
  { valor: 'golpe', rotulo: 'Suspeita de golpe' },
  { valor: 'organizador', rotulo: 'Sou organizador' },
  { valor: 'repasse', rotulo: 'Repasse e financeiro' },
  { valor: 'outro', rotulo: 'Outro assunto' },
]

export type Verificacao = {
  situacao: 'valido' | 'utilizado' | 'cancelado' | 'na_revenda' | 'transferido' | 'nao_pago' | 'nao_encontrado' | 'evento_encerrado'
  evento_titulo?: string
  evento_slug?: string
  evento_inicio_em?: string
  tipo?: string
}

export type ComponenteStatus = { chave: string; nome: string; status: 'operacional' | 'degradado' | 'falha'; detalhe?: string }
export type AvisoStatus = { id: number; titulo: string; mensagem: string; nivel: 'aviso' | 'falha' | 'manutencao'; componente: string; inicio_em: string; fim_em: string | null }
export type StatusPlataforma = {
  geral: ComponenteStatus['status']
  componentes: ComponenteStatus[]
  avisos: AvisoStatus[]
  recentes: AvisoStatus[]
  atualizado_em: string
}
