-- Uploads guardados no próprio banco (capa, fotos da galeria, áudio...).
-- Em produção o disco do servidor é apagado a cada deploy (a capa sumia);
-- o banco não. Quando houver um bucket (Cloudflare R2/S3), STORAGE_DRIVER=s3
-- passa a gravar lá e esta tabela só serve os arquivos antigos.
CREATE TABLE arquivos (
    nome         VARCHAR(120) PRIMARY KEY,
    content_type VARCHAR(80)  NOT NULL,
    tamanho      INT          NOT NULL,
    conteudo     BYTEA        NOT NULL,
    criado_em    TIMESTAMPTZ  NOT NULL DEFAULT now()
);
