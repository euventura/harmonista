/**
 * TinyMDE Manager - Sistema modularizado para editor Markdown
 * Funcionalidades:
 * - Modo tela cheia
 * - Salvamento automático (apenas rascunhos)
 * - Indicador visual de alterações
 */

class TinyMDEManager {
    constructor(options = {}) {
        this.textareaId = options.textareaId || 'txt_content';
        this.titleId = options.titleId || 'title';
        this.isDraftField = options.isDraftField || 'draft';
        this.autoSaveUrl = options.autoSaveUrl || null;
        this.autoSaveInterval = options.autoSaveInterval || 30000; // 30 segundos
        
        this.editor = null;
        this.autoSaveTimer = null;
        this.isFullscreen = false;
        this.originalContent = '';
        this.hasChanges = false;
        
        this.init();
    }
    
    init() {
        this.createEditor();
        this.addEventListeners();
        this.createFullscreenButton();
        this.createStatusIndicator();
        this.startAutoSave();
    }
    
    createEditor() {
        this.editor = new TinyMDE.Editor({
            textarea: this.textareaId,
            renderMarkdown: false,
            autoComplete: false,
            tabSize: 2
        });
        
        // Event listeners para renderização
        const textarea = document.getElementById(this.textareaId);
        
        textarea.addEventListener('blur', () => {
            if (this.editor.render) {
                this.editor.render();
            }
        });
        
        textarea.addEventListener('keydown', (e) => {
            if (e.ctrlKey && e.key === 'r') {
                e.preventDefault();
                if (this.editor.render) {
                    this.editor.render();
                }
            }
            
            // F11 para tela cheia
            if (e.key === 'F11') {
                e.preventDefault();
                this.toggleFullscreen();
            }
        });
        
        // Detectar mudanças
        textarea.addEventListener('input', () => {
            this.markAsChanged();
        });
        
        // Salvar conteúdo original
        this.originalContent = textarea.value;
    }
    
    addEventListeners() {
        // Escapar do modo fullscreen com ESC
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape' && this.isFullscreen) {
                this.exitFullscreen();
            }
        });
    }
    
    createFullscreenButton() {
        const textarea = document.getElementById(this.textareaId);
        const container = textarea.parentElement;
        
        const button = document.createElement('button');
        button.type = 'button';
        button.className = 'tinymde-fullscreen-btn';
        button.innerHTML = '⛶ Tela Cheia (F11)';
        button.onclick = () => this.toggleFullscreen();
        
        // Adicionar CSS inline
        button.style.cssText = `
            position: absolute;
            top: 5px;
            right: 5px;
            background: var(--accent, #0969da);
            color: white;
            border: none;
            padding: 4px 8px;
            border-radius: 4px;
            font-size: 12px;
            cursor: pointer;
            z-index: 10;
        `;
        
        container.style.position = 'relative';
        container.appendChild(button);
    }
    
    createStatusIndicator() {
        const textarea = document.getElementById(this.textareaId);
        const container = textarea.parentElement;
        
        const indicator = document.createElement('div');
        indicator.id = 'tinymde-status';
        indicator.style.cssText = `
            position: absolute;
            top: -25px;
            left: 0;
            background: #28a745;
            color: white;
            padding: 2px 8px;
            border-radius: 3px;
            font-size: 12px;
            display: none;
            z-index: 10;
        `;
        
        container.appendChild(indicator);
    }
    
    toggleFullscreen() {
        if (this.isFullscreen) {
            this.exitFullscreen();
        } else {
            this.enterFullscreen();
        }
    }
    
    enterFullscreen() {
        const textarea = document.getElementById(this.textareaId);
        const titleInput = document.getElementById(this.titleId);
        const main = document.querySelector('main');
        
        // Salvar estado original
        this.originalState = {
            mainDisplay: main.style.display,
            mainChildren: Array.from(main.children),
            textareaStyle: textarea.style.cssText,
            titleStyle: titleInput ? titleInput.style.cssText : ''
        };
        
        // Limpar main e adicionar apenas título e textarea
        main.innerHTML = '';
        main.style.cssText = `
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: var(--bg-default, #fff);
            z-index: 1000;
            padding: 20px;
            display: flex;
            flex-direction: column;
        `;
        
        // Criar container fullscreen
        const container = document.createElement('div');
        container.style.cssText = `
            display: flex;
            flex-direction: column;
            height: 100%;
            gap: 10px;
        `;
        
        // Título em fullscreen
        if (titleInput) {
            const titleClone = titleInput.cloneNode(true);
            titleClone.style.cssText = `
                font-size: 24px;
                padding: 10px;
                border: 2px solid var(--accent, #0969da);
                border-radius: 6px;
            `;
            container.appendChild(titleClone);
            
            // Sincronizar valores
            titleClone.addEventListener('input', () => {
                titleInput.value = titleClone.value;
            });
            titleInput.addEventListener('input', () => {
                titleClone.value = titleInput.value;
            });
        }
        
        // Textarea em fullscreen
        const textareaClone = textarea.cloneNode(true);
        textareaClone.style.cssText = `
            flex: 1;
            font-family: 'Courier New', monospace;
            font-size: 16px;
            padding: 15px;
            border: 2px solid var(--accent, #0969da);
            border-radius: 6px;
            resize: none;
        `;
        container.appendChild(textareaClone);
        
        // Sincronizar valores
        textareaClone.addEventListener('input', () => {
            textarea.value = textareaClone.value;
            this.markAsChanged();
        });
        textarea.addEventListener('input', () => {
            textareaClone.value = textarea.value;
        });
        
        // Botão sair
        const exitBtn = document.createElement('button');
        exitBtn.innerHTML = '✕ Sair da Tela Cheia (ESC)';
        exitBtn.style.cssText = `
            position: absolute;
            top: 10px;
            right: 10px;
            background: var(--danger, #dc3545);
            color: white;
            border: none;
            padding: 8px 12px;
            border-radius: 4px;
            cursor: pointer;
        `;
        exitBtn.onclick = () => this.exitFullscreen();
        
        main.appendChild(container);
        main.appendChild(exitBtn);
        
        // Focar no textarea
        setTimeout(() => textareaClone.focus(), 100);
        
        this.isFullscreen = true;
        this.fullscreenTextarea = textareaClone;
    }
    
    exitFullscreen() {
        if (!this.isFullscreen) return;
        
        const main = document.querySelector('main');
        
        // Restaurar estado original
        main.style.cssText = this.originalState.mainStyle || '';
        main.innerHTML = '';
        
        // Restaurar filhos originais
        this.originalState.mainChildren.forEach(child => {
            main.appendChild(child);
        });
        
        this.isFullscreen = false;
        this.fullscreenTextarea = null;
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
        const indicator = document.getElementById('tinymde-status');
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
        
        const textarea = document.getElementById(this.textareaId);
        const titleInput = document.getElementById(this.titleId);
        
        const data = {
            title: titleInput ? titleInput.value : '',
            content: textarea.value,
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
        const textarea = document.getElementById(this.textareaId);
        if (textarea) {
            textarea.value = content;
            this.originalContent = content;
            if (this.editor && this.editor.setContent) {
                this.editor.setContent(content);
            }
        }
    }
    
    destroy() {
        if (this.autoSaveTimer) {
            clearInterval(this.autoSaveTimer);
        }
        
        if (this.isFullscreen) {
            this.exitFullscreen();
        }
    }
}

// Função global para inicializar facilmente
window.initTinyMDE = function(options = {}) {
    // Aguardar o DOM carregar
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', () => {
            return new TinyMDEManager(options);
        });
    } else {
        return new TinyMDEManager(options);
    }
};