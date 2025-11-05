/**
 * EasyMDE Manager - Sistema modularizado para editor Markdown
 * Funcionalidades:
 * - Modo tela cheia (nativo do EasyMDE)
 * - Salvamento automático (apenas rascunhos)
 * - Indicador visual de alterações
 * - Toolbar customizável
 */

class EasyMDEManager {
    constructor(options = {}) {
        this.textareaId = options.textareaId || 'txt_content';
        this.titleId = options.titleId || 'title';
        this.isDraftField = options.isDraftField || 'draft';
        this.autoSaveUrl = options.autoSaveUrl || null;
        this.autoSaveInterval = options.autoSaveInterval || 30000; // 30 segundos

        this.editor = null;
        this.autoSaveTimer = null;
        this.originalContent = '';
        this.hasChanges = false;

        this.init();
    }

    init() {
        this.createEditor();
        this.createStatusIndicator();
        this.startAutoSave();
    }

    createEditor() {
        const textarea = document.getElementById(this.textareaId);
        if (!textarea) {
            console.error(`Textarea com ID '${this.textareaId}' não encontrado`);
            return;
        }

        // Configuração do EasyMDE
        this.editor = new EasyMDE({
            element: textarea,
            autoDownloadFontAwesome: true,
            autofocus: false,
            hideIcons: ['side-by-side'],
            indentWithTabs: false,
            tabSize: 2,
            lineNumbers: false,
            lineWrapping: true,
            minHeight: '300px',
            placeholder: 'Digite seu conteúdo em Markdown...',
            spellChecker: false,
            status: ['lines', 'words', 'cursor'], // Barra de status
            styleSelectedText: true,

            // Toolbar customizada
            toolbar: [
                'bold',
                'italic',
                'heading',
                '|',
                'quote',
                'unordered-list',
                'ordered-list',
                '|',
                'link',
                'image',
                '|',
                'preview',
                'side-by-side',
                'fullscreen',
                '|',
                'guide'
            ],

            // Atalhos de teclado
            shortcuts: {
                toggleFullScreen: 'F11',
                togglePreview: 'Ctrl-R',
                toggleSideBySide: null
            }
        });

        // Salvar conteúdo original
        this.originalContent = this.editor.value();

        // Event listener para detectar mudanças
        this.editor.codemirror.on('change', () => {
            this.markAsChanged();
        });

        // Sincronizar com textarea original (para compatibilidade com formulários)
        this.editor.codemirror.on('change', () => {
            textarea.value = this.editor.value();
        });
    }

    createStatusIndicator() {
        const editorToolbar = document.querySelector('.EasyMDEContainer .editor-toolbar');
        if (!editorToolbar) return;

        const indicator = document.createElement('div');
        indicator.id = 'easymde-status';
        indicator.style.cssText = `
            position: absolute;
            top: 40px;
            right: 10px;
            background: #28a745;
            color: white;
            padding: 4px 10px;
            border-radius: 4px;
            font-size: 12px;
            display: none;
            z-index: 10;
            box-shadow: 0 2px 4px rgba(0,0,0,0.2);
        `;

        editorToolbar.parentElement.style.position = 'relative';
        editorToolbar.parentElement.appendChild(indicator);
    }

    markAsChanged() {
        if (!this.hasChanges) {
            this.hasChanges = true;
            this.showStatus('Alterações não salvas', '#ffc107');
        }
    }

    markAsSaved() {
        this.hasChanges = false;
        this.showStatus('Alterações salvas automaticamente', '#28a745');
    }

    showStatus(message, color = '#28a745') {
        const indicator = document.getElementById('easymde-status');
        if (indicator) {
            indicator.textContent = message;
            indicator.style.backgroundColor = color;
            indicator.style.display = 'block';

            // Esconder após 3 segundos
            setTimeout(() => {
                indicator.style.display = 'none';
            }, 3000);
        }
    }

    startAutoSave() {
        if (!this.autoSaveUrl) return;

        this.autoSaveTimer = setInterval(() => {
            this.performAutoSave();
        }, this.autoSaveInterval);
    }

    async performAutoSave() {
        // Só salvar se for rascunho e tiver mudanças
        const draftField = document.getElementById(this.isDraftField);
        const isDraft = draftField && draftField.value === 'true';

        if (!isDraft || !this.hasChanges) return;

        const titleInput = document.getElementById(this.titleId);

        const data = {
            title: titleInput ? titleInput.value : '',
            content: this.editor.value(),
            draft: true,
            auto_save: true
        };

        try {
            const response = await fetch(this.autoSaveUrl, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(data)
            });

            if (response.ok) {
                this.markAsSaved();
            } else {
                this.showStatus('Erro ao salvar automaticamente', '#dc3545');
            }
        } catch (error) {
            this.showStatus('Erro de conexão no auto-save', '#dc3545');
        }
    }

    setContent(content) {
        if (this.editor) {
            this.editor.value(content);
            this.originalContent = content;
        }
    }

    getValue() {
        return this.editor ? this.editor.value() : '';
    }

    destroy() {
        if (this.autoSaveTimer) {
            clearInterval(this.autoSaveTimer);
        }

        if (this.editor) {
            this.editor.toTextArea();
            this.editor = null;
        }
    }
}

// Função global para inicializar facilmente
window.initEasyMDE = function(options = {}) {
    // Aguardar o DOM e o EasyMDE carregarem
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', () => {
            return new EasyMDEManager(options);
        });
    } else {
        return new EasyMDEManager(options);
    }
};
