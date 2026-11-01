import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'

import { VoltarLink } from '@/components/voltar-link'
import { api, ApiError } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'

type Chave = { id: number; nome: string; prefixo: string; ativo: boolean; criado_em: string; ultimo_uso_em: string | null }

export default function ApiKeys() {
  const queryClient = useQueryClient()
  const [nome, setNome] = useState('')
  const [novaChave, setNovaChave] = useState<string | null>(null)
  const [erro, setErro] = useState<string | null>(null)
  const { data, error, isPending } = useQuery({ queryKey: ['org-api-keys'], queryFn: () => api<Chave[]>('/org/api-keys') })
  const invalidar = () => queryClient.invalidateQueries({ queryKey: ['org-api-keys'] })

  const criar = async () => {
    setErro(null)
    try {
      const r = await api<{ chave: string }>('/org/api-keys', { method: 'POST', body: { nome } })
      setNovaChave(r.chave)
      setNome('')
      invalidar()
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao criar chave')
    }
  }

  const revogar = async (id: number) => {
    if (!confirm('Revogar esta chave? Integrações que a usam vão parar de funcionar.')) return
    setErro(null)
    try {
      await api(`/org/api-keys/${id}`, { method: 'DELETE' })
      invalidar()
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível revogar a chave. Tente novamente.')
    }
  }

  if (error instanceof ApiError && error.status === 403) {
    return <p className="p-8 text-center text-muted-foreground">Só o dono do perfil de organizador gerencia chaves de API.</p>
  }

  return (
    <main className="mx-auto max-w-3xl px-4 py-12">
      <VoltarLink to="/organizador/eventos">Voltar ao painel</VoltarLink>
      <h1 className="mb-2 text-2xl font-semibold text-foreground">Integrações e API</h1>
      <p className="mb-6 text-muted-foreground">
        Conecte informações dos seus eventos a outro sistema. Aqui você cria a chave que permite essa conexão e acompanha quais chaves já estão em uso.
      </p>

      <Card className="mb-6">
        <CardContent className="space-y-4 py-6 text-sm">
          <h2 className="text-lg font-semibold text-foreground">Primeira vez por aqui?</h2>
          <p className="text-muted-foreground">Uma API é uma forma de dois sistemas conversarem. Por exemplo: um painel da sua equipe pode consultar a Evve para mostrar quantos ingressos estão disponíveis, sem alguém precisar copiar os números manualmente.</p>
          <p className="text-muted-foreground">A chave de API funciona como uma credencial de acesso para o sistema conectado. Criar uma chave é apenas o primeiro passo: a conexão precisa ser configurada por quem desenvolve ou administra esse sistema.</p>
          <div className="rounded-lg bg-muted p-4 text-foreground">
            <p className="font-medium">Preciso configurar isso para usar a Evve?</p>
            <p className="mt-1 text-muted-foreground">Não. Você pode criar eventos, vender ingressos e acompanhar tudo pelo painel normalmente. Use esta área quando precisar levar informações para outro sistema.</p>
          </div>
          <p className="text-muted-foreground"><strong className="text-foreground">O acesso é somente para consulta:</strong> a integração pode ler informações dos eventos do seu perfil de organizador, mas não pode criar ou editar eventos, vender ingressos ou registrar entradas.</p>
        </CardContent>
      </Card>

      <section className="mb-6" aria-labelledby="passos-api">
        <h2 id="passos-api" className="mb-3 text-lg font-semibold text-foreground">Como começar, passo a passo</h2>
        <ol className="list-decimal space-y-3 pl-5 text-sm text-muted-foreground">
          <li><strong className="text-foreground">Defina o que vai conectar.</strong> Combine com a pessoa responsável pelo outro sistema quais informações precisam ser consultadas.</li>
          <li><strong className="text-foreground">Crie uma chave abaixo.</strong> Dê um nome fácil de reconhecer, como “Painel da equipe”. Prefira uma chave para cada integração, para poder interromper uma conexão sem afetar as outras.</li>
          <li><strong className="text-foreground">Guarde a chave assim que ela aparecer.</strong> Ela só é exibida uma vez. Passe-a por um canal seguro à pessoa responsável pela configuração.</li>
          <li><strong className="text-foreground">Configure e teste a consulta.</strong> O outro sistema usa a chave para pedir os dados à Evve. Depois do teste, confira o “Último uso” na lista abaixo.</li>
        </ol>
      </section>

      <section className="mb-6 rounded-xl border border-border p-5" aria-labelledby="criar-chave-api">
        <h2 id="criar-chave-api" className="text-lg font-semibold text-foreground">Criar uma chave de acesso</h2>
        <p className="mt-1 mb-4 text-sm text-muted-foreground">Cada chave dá acesso às consultas dos seus eventos, incluindo nomes e e-mails de titulares de ingressos. Não é possível limitar uma chave a apenas um evento ou tipo de informação.</p>
        <form onSubmit={(event) => { event.preventDefault(); if (nome.trim()) void criar() }} className="space-y-2">
          <label htmlFor="nome-chave-api" className="text-sm font-medium text-foreground">Nome da integração</label>
          <div className="flex flex-col gap-2 sm:flex-row">
            <Input id="nome-chave-api" aria-describedby="ajuda-nome-chave" placeholder="Ex.: Painel da equipe" value={nome} onChange={(e) => setNome(e.target.value)} />
            <Button type="submit" disabled={!nome.trim()}>Criar chave</Button>
          </div>
          <p id="ajuda-nome-chave" className="text-xs text-muted-foreground">Esse nome serve para você identificar a conexão. Ele não é a chave de acesso.</p>
        </form>
      </section>

      {novaChave && (
        <Card className="mb-4 border-primary">
          <CardContent className="py-4">
            <p role="status" className="text-sm font-medium text-foreground">Chave criada! Copie e guarde agora: ela não será exibida de novo.</p>
            <p className="mt-1 font-mono text-xs break-all text-foreground">{novaChave}</p>
            <p className="mt-3 text-sm text-muted-foreground">Guarde como uma senha. Se perder a chave, crie outra, atualize a integração e revogue a antiga.</p>
          </CardContent>
        </Card>
      )}

      {erro && <p role="alert" className="mb-4 text-sm text-destructive">{erro}</p>}
      <section aria-labelledby="suas-chaves-api" className="mb-6">
      <h2 id="suas-chaves-api" className="mb-2 text-lg font-semibold text-foreground">Suas chaves de acesso</h2>
      <p className="mb-4 text-sm text-muted-foreground">“Último uso” mostra a consulta mais recente feita com a chave. Revogar bloqueia novas consultas imediatamente e não pode ser desfeito.</p>
      <div className="flex flex-col gap-2">
        {isPending && <p role="status" className="text-sm text-muted-foreground">Carregando suas chaves…</p>}
        {error && <p role="alert" className="text-sm text-destructive">Não foi possível carregar suas chaves. Atualize a página para tentar novamente.</p>}
        {data?.length === 0 && <p className="rounded-lg bg-muted p-4 text-sm text-muted-foreground">Você ainda não criou uma chave. Quando precisar conectar outro sistema, use o formulário acima.</p>}
        {data?.map((k) => (
          <Card key={k.id}>
            <CardContent className="flex flex-wrap items-center justify-between gap-3 py-4">
              <div className="min-w-0 break-words">
                <p className="text-sm font-medium text-foreground">
                  {k.nome} <span className="text-xs text-muted-foreground">({k.ativo ? 'ativa' : 'revogada'})</span>
                </p>
                <p className="font-mono text-xs text-muted-foreground">
                  Início da chave: {k.prefixo}…
                </p>
                <p className="text-xs text-muted-foreground">Último uso: {k.ultimo_uso_em ? new Date(k.ultimo_uso_em).toLocaleString('pt-BR') : 'nenhuma consulta registrada'}</p>
              </div>
              {k.ativo && (
                <Button variant="destructive" size="sm" aria-label={`Revogar chave ${k.nome}`} onClick={() => revogar(k.id)}>
                  Revogar
                </Button>
              )}
            </CardContent>
          </Card>
        ))}
      </div>
      </section>

      <details className="rounded-xl border border-border p-5 text-sm">
        <summary className="cursor-pointer font-semibold text-foreground">Informações técnicas para configurar a integração</summary>
        <div className="mt-4 space-y-4 text-muted-foreground">
          <p>Use a chave apenas no servidor do sistema conectado. Não coloque a chave no código de páginas públicas ou de aplicativos distribuídos aos usuários.</p>
          <p>Faça uma requisição <code>GET</code> (consulta) e envie a chave no cabeçalho <code>X-API-Key</code>, um campo de identificação que acompanha a requisição. A resposta vem em JSON, um formato de dados que o outro sistema consegue ler.</p>
          <p>Comece pela lista de eventos. Use o campo <code>id</code> de um evento retornado para substituir <code>:id</code> nas outras consultas.</p>
          <dl className="space-y-3">
            {[
              ['/eventos', 'Lista os eventos do seu perfil de organizador e seus identificadores.'],
              ['/eventos/:id', 'Consulta título, datas, categoria e status de um evento.'],
              ['/eventos/:id/ingressos', 'Consulta tipos de ingresso, preços em centavos e quantidades ocupadas e disponíveis.'],
              ['/eventos/:id/participantes', 'Consulta titulares de ingressos pagos ou utilizados: nome, e-mail, código e status. Não é a lista de talentos inscritos em concursos.'],
              ['/eventos/:id/resumo', 'Consulta os contadores de ingressos pagos e utilizados no check-in.'],
            ].map(([rota, descricao]) => (
              <div key={rota}>
                <dt className="break-all font-mono text-xs text-foreground">GET /api/public/v1{rota}</dt>
                <dd className="mt-1">{descricao}</dd>
              </div>
            ))}
          </dl>
          <p>Exemplo com chave fictícia: troque o endereço pelo domínio da sua instalação e <code>SUA_CHAVE</code> pela chave guardada no servidor.</p>
          <pre className="overflow-x-auto rounded-lg bg-muted p-3 text-xs text-foreground">{'curl "https://SEU_DOMINIO/api/public/v1/eventos" -H "X-API-Key: SUA_CHAVE"'}</pre>
          <p>A integração precisa fazer uma nova consulta para atualizar os dados. Esta API não envia notificações automáticas (webhooks).</p>
          <p>Se houver erro: <strong>401</strong> indica chave ausente, inválida ou revogada; <strong>404</strong> indica evento não encontrado ou fora do seu perfil; <strong>429</strong> indica excesso de consultas — aguarde antes de tentar novamente.</p>
        </div>
      </details>
    </main>
  )
}
