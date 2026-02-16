// ============================================================================
// Utilities
// ============================================================================

function element_get(id: string): HTMLElement {
    const element = document.getElementById(id);
    if (!element) throw new Error(`Element #${id} not found`);
    return element;
}

function element_get_or_null(id: string): HTMLElement | null {
    return document.getElementById(id);
}

// ============================================================================
// Toast Notifications
// ============================================================================

type ToastType = 'success' | 'error' | 'warning';

function toast_show(message: string, type: ToastType = 'success', duration: number = 3000): void {
    let container = document.getElementById('toast-container');
    if (!container) {
        container = document.createElement('div');
        container.id = 'toast-container';
        container.className = 'fixed top-4 right-4 z-50 flex flex-col gap-2';
        document.body.appendChild(container);
    }
    
    const colors: Record<ToastType, string> = {
        success: 'bg-green-500',
        error: 'bg-red-500',
        warning: 'bg-yellow-500',
    };
    
    const toast = document.createElement('div');
    toast.className = `${colors[type]} text-white px-4 py-2 rounded-lg shadow-lg transform transition-all duration-300 translate-x-full opacity-0`;
    toast.textContent = message;
    
    container.appendChild(toast);
    
    requestAnimationFrame(() => {
        toast.classList.remove('translate-x-full', 'opacity-0');
    });
    
    setTimeout(() => {
        toast.classList.add('translate-x-full', 'opacity-0');
        setTimeout(() => toast.remove(), 300);
    }, duration);
}

// ============================================================================
// Document Level: Close dropdowns on outside click
// ============================================================================

function document_click_outside_init(): void {
    document.addEventListener('click', (e) => {
        const target = e.target as Node;
        
        const fixed = [
            { container: 'year-select-container', dropdown: 'year-dropdown' },
            { container: 'user-menu-container', dropdown: 'user-menu-dropdown' },
        ];
        
        fixed.forEach(({ container, dropdown }) => {
            const c = document.getElementById(container);
            const d = document.getElementById(dropdown);
            if (c && d && !c.contains(target)) {
                d.classList.add('hidden');
            }
        });
        
        document.querySelectorAll('[data-enum-container]').forEach(container => {
            if (!container.contains(target)) {
                container.querySelector('[data-enum-dropdown]')?.classList.add('hidden');
            }
        });
    });
}

// ============================================================================
// Year Select
// ============================================================================

type StateYearSelect = {
    container: HTMLElement;
    button: HTMLElement;
    dropdown: HTMLElement;
};

function year_select_init(): StateYearSelect | null {
    const container = element_get_or_null('year-select-container');
    if (!container) return null;

    const state: StateYearSelect = {
        container,
        button: element_get('year-select-button'),
        dropdown: element_get('year-dropdown'),
    };

    state.button.addEventListener('click', (e: Event) => {
        e.stopPropagation();
        state.dropdown.classList.toggle('hidden');
    });

    return state;
}

// ============================================================================
// User Menu
// ============================================================================

type StateUserMenu = {
    container: HTMLElement;
    button: HTMLElement;
    dropdown: HTMLElement;
    is_click_open: boolean;
    hover_timeout: number | null;
};

function user_menu_open(state: StateUserMenu): void {
    state.dropdown.classList.remove('hidden');
    state.button.setAttribute('aria-expanded', 'true');
}

function user_menu_close(state: StateUserMenu): void {
    state.dropdown.classList.add('hidden');
    state.button.setAttribute('aria-expanded', 'false');
    state.is_click_open = false;
}

function user_menu_click(state: StateUserMenu, e: Event): void {
    e.stopPropagation();
    
    if (state.is_click_open) {
        user_menu_close(state);
    } else {
        user_menu_open(state);
        state.is_click_open = true;
    }
}

function user_menu_mouse_enter(state: StateUserMenu): void {
    if (state.hover_timeout !== null) {
        clearTimeout(state.hover_timeout);
        state.hover_timeout = null;
    }
    if (!state.is_click_open) {
        user_menu_open(state);
    }
}

function user_menu_mouse_leave(state: StateUserMenu): void {
    if (!state.is_click_open) {
        state.hover_timeout = window.setTimeout(() => user_menu_close(state), 200);
    }
}

function user_menu_init(): StateUserMenu | null {
    const container = element_get_or_null('user-menu-container');
    if (!container) return null;
    
    const state: StateUserMenu = {
        container,
        button: element_get('user-menu-button'),
        dropdown: element_get('user-menu-dropdown'),
        is_click_open: false,
        hover_timeout: null,
    };
    
    state.button.addEventListener('click', (e) => user_menu_click(state, e));
    state.container.addEventListener('mouseenter', () => user_menu_mouse_enter(state));
    state.container.addEventListener('mouseleave', () => user_menu_mouse_leave(state));
    
    return state;
}

// ============================================================================
// Logout
// ============================================================================

type StateLogout = {
    button: HTMLElement;
    progress: HTMLElement;
    form: HTMLFormElement;
    timer: number | null;
    start_time: number | null;
    hold_duration: number;
};

function logout_update(state: StateLogout): void {
    if (state.start_time === null) return;
    
    const elapsed = Date.now() - state.start_time;
    const progress = Math.min((elapsed / state.hold_duration) * 100, 100);
    state.progress.style.width = `${progress}%`;
    
    if (progress >= 100) {
        state.form.submit();
    } else {
        state.timer = requestAnimationFrame(() => logout_update(state));
    }
}

function logout_cancel(state: StateLogout): void {
    if (state.timer !== null) {
        cancelAnimationFrame(state.timer);
        state.timer = null;
    }
    state.start_time = null;
    state.progress.style.width = '0%';
}

function logout_start(state: StateLogout, e: Event): void {
    e.preventDefault();
    state.start_time = Date.now();
    logout_update(state);
}

function logout_init(): StateLogout | null {
    const button = element_get_or_null('logout-button');
    if (!button) return null;
    
    const state: StateLogout = {
        button,
        progress: element_get('logout-progress'),
        form: element_get('logout-form') as HTMLFormElement,
        timer: null,
        start_time: null,
        hold_duration: 750,
    };
    
    state.button.addEventListener('mousedown', (e) => logout_start(state, e));
    state.button.addEventListener('mouseup', () => logout_cancel(state));
    state.button.addEventListener('mouseleave', () => logout_cancel(state));
    
    return state;
}

// ============================================================================
// Session Timer
// ============================================================================

type StateSessionTimer = {
    display: HTMLElement;
    timeout_seconds: number;
    last_activity: number;
    interval: number | null;
};

function session_timer_update(state: StateSessionTimer): void {
    const elapsed = Math.floor((Date.now() - state.last_activity) / 1000);
    const remaining = Math.max(0, state.timeout_seconds - elapsed);
    
    const minutes = Math.floor(remaining / 60);
    const seconds = remaining % 60;
    state.display.textContent = `${minutes}:${seconds.toString().padStart(2, '0')}`;
    
    if (remaining <= 0) {
        window.location.href = '/logout';
    }
}

function session_timer_reset(state: StateSessionTimer): void {
    state.last_activity = Date.now();
}

function session_timer_init(timeout_minutes: number): StateSessionTimer | null {
    const display = element_get_or_null('session-timer');
    if (!display) return null;
    
    const state: StateSessionTimer = {
        display,
        timeout_seconds: timeout_minutes * 60,
        last_activity: Date.now(),
        interval: null,
    };
    
    document.addEventListener('keypress', () => session_timer_reset(state), true);
    document.addEventListener('click', () => session_timer_reset(state), true);
    
    state.interval = window.setInterval(() => session_timer_update(state), 1000);
    session_timer_update(state);
    
    return state;
}

// ============================================================================
// Nav Toggle
// ============================================================================

type StateNavToggle = {
    nav: HTMLElement;
    toggle: HTMLElement;
    icon_menu: HTMLElement;
    icon_close: HTMLElement;
    is_expanded: boolean;
};

function nav_toggle_set(state: StateNavToggle, expanded: boolean): void {
    state.is_expanded = expanded;
    
    state.nav.classList.toggle('w-16', !expanded);
    state.nav.classList.toggle('w-64', expanded);
    state.icon_menu.classList.toggle('hidden', expanded);
    state.icon_close.classList.toggle('hidden', !expanded);
    
    document.querySelectorAll('.nav-text').forEach(el => {
        el.classList.toggle('hidden', !expanded);
    });
}

function nav_toggle_init(): StateNavToggle | null {
    const nav = element_get_or_null('left-nav');
    if (!nav) return null;
    
    const state: StateNavToggle = {
        nav,
        toggle: element_get('nav-toggle'),
        icon_menu: element_get('icon-menu'),
        icon_close: element_get('icon-close'),
        is_expanded: false,
    };
    
    state.toggle.addEventListener('click', () => nav_toggle_set(state, !state.is_expanded));
    
    return state;
}

// ============================================================================
// Number Formatting
// ============================================================================

type NumberFormat = {
    has_spaces: boolean;
    decimals: number;
};

function number_format_parse(format: string): NumberFormat {
    const has_spaces = format.includes(' ');
    let decimal_part = '';
    
    if (format.includes('.')) {
        decimal_part = format.split('.')[1] ?? '';
    } else if (format.includes(',')) {
        decimal_part = format.split(',')[1] ?? '';
    }
    
    return { has_spaces, decimals: decimal_part.length };
}

function number_format_value(value: number, fmt: NumberFormat): string {
    let result = fmt.decimals === 0 
        ? Math.round(value).toString() 
        : value.toFixed(fmt.decimals);
    
    result = result.replace('.', ',');
    
    if (fmt.has_spaces) {
        const parts = result.split(',');
        const int_part = parts[0] ?? '';
        const dec_part = parts[1];
        
        const is_negative = int_part.startsWith('-');
        const abs_int = is_negative ? int_part.substring(1) : int_part;
        const spaced = abs_int.replace(/\B(?=(\d{3})+(?!\d))/g, ' ');
        result = (is_negative ? '-' : '') + spaced + (dec_part ? ',' + dec_part : '');
    }
    
    return result;
}

function number_value_parse(value: string): number | null {
    const raw = value.replace(/\s/g, '').replace(',', '.');
    if (raw === '' || raw === '-' || raw === '.') return null;
    const num = parseFloat(raw);
    return isNaN(num) ? null : num;
}

function string_is_blank(value: string): boolean {    
    // return value.replace(/\s/g, '').replace(',', '').replace('.', '').replace('-', '') === '';
    return value.replace(/[\s,.\-]/g, '') === ''
}

// ============================================================================
// Row Selection Helper
// ============================================================================

function row_cells_get(table: HTMLElement, rowIndex: number): HTMLElement[] {
    return Array.from(table.querySelectorAll<HTMLElement>(`[data-cell][data-row-index="${rowIndex}"]`));
}

function row_index_get(cell: HTMLElement): number | null {
    const index = cell.dataset.rowIndex;
    return index !== undefined ? parseInt(index, 10) : null;
}

function all_row_indices_get(state: StateTable): number[] {
    const indices = new Set<number>();
    state.element.querySelectorAll<HTMLElement>('[data-cell][data-row-index]').forEach(cell => {
        const idx = row_index_get(cell);
        if (idx !== null) indices.add(idx);
    });
    return Array.from(indices).sort((a, b) => a - b);
}

// ============================================================================
// Input Error Display
// ============================================================================

type ErrorType = 'error' | 'warning';

function input_show_error(input: HTMLInputElement, message: string, type: ErrorType = 'error'): void {
    input_clear_error(input);
    
    input.dataset.errorMessage = message;
    input.dataset.errorType = type;
    
    input.classList.remove('border-gray-200', 'focus:border-indigo-500');
    
    if (type === 'error') {
        input.classList.add('border-red-300', 'bg-red-50', 'ring-2', 'ring-red-200');
    } else {
        input.classList.add('border-orange-300', 'bg-orange-50', 'ring-2', 'ring-orange-200');
    }
}

function input_show_error_popup(input: HTMLInputElement): void {
    const message = input.dataset.errorMessage;
    const type = input.dataset.errorType as ErrorType || 'error';
    if (!message) return;
    
    const wrapper = input.closest('[data-cell]') as HTMLElement;
    if (!wrapper) return;
    
    wrapper.querySelector('.input-error-popup')?.remove();
    wrapper.style.position = 'relative';
    
    const isError = type === 'error';
    const borderColor = isError ? 'border-red-300' : 'border-orange-300';
    const bgColor = isError ? 'bg-red-500' : 'bg-orange-500';
    const textColor = isError ? 'text-red-700' : 'text-orange-700';
    
    const popup = document.createElement('div');
    popup.className = `input-error-popup absolute left-0 top-full mt-1 z-50 bg-white border ${borderColor} rounded-lg shadow-lg p-2 min-w-max`;
    popup.innerHTML = `
        <div class="flex items-start gap-2">
            <div class="shrink-0 w-5 h-5 ${bgColor} rounded-full flex items-center justify-center">
                <svg class="w-3 h-3 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M6 18L18 6M6 6l12 12"/>
                </svg>
            </div>
            <p class="text-sm ${textColor} font-medium">${message}</p>
        </div>
    `;
    
    wrapper.appendChild(popup);
}

function input_clear_error(input: HTMLInputElement): void {
    delete input.dataset.errorMessage;
    delete input.dataset.errorType;
    
    const wrapper = input.closest('[data-cell]') as HTMLElement;
    wrapper?.querySelector('.input-error-popup')?.remove();
    
    input.classList.remove(
        'border-red-300', 'bg-red-50', 'ring-2', 'ring-red-200',
        'border-orange-300', 'bg-orange-50', 'ring-orange-200'
    );
    input.classList.add('border-gray-200', 'focus:border-indigo-500');
}

// ============================================================================
// Input Success Display
// ============================================================================

function input_show_success(input: HTMLInputElement): void {
    input.classList.remove('border-gray-200', 'focus:border-indigo-500');
    input.classList.add('border-green-300', 'bg-green-50', 'ring-2', 'ring-green-200');
    
    setTimeout(() => {
        input.classList.remove('border-green-300', 'bg-green-50', 'ring-2', 'ring-green-200');
        input.classList.add('border-gray-200', 'focus:border-indigo-500');
    }, 2000);
}

// ============================================================================
// Row-Based Operations (using row index)
// ============================================================================

function row_cells_has_data(cells: HTMLElement[]): boolean {
    for (const cell of cells) {
        const numberInputs = cell.querySelectorAll<HTMLInputElement>('.number-input:not([readonly])');
        for (const input of numberInputs) {
            if (input.value.trim() !== '') return true;
        }
        
        const stringInputs = cell.querySelectorAll<HTMLInputElement>('.string-input:not([readonly])');
        for (const input of stringInputs) {
            if (input.value.trim() !== '') return true;
        }
        
        const enumInputs = cell.querySelectorAll<HTMLInputElement>('[data-enum-value]');
        for (const input of enumInputs) {
            const container = input.closest('[data-enum-container]');
            const visibleInput = container?.querySelector('[data-enum-input]') as HTMLInputElement;
            if (visibleInput?.hasAttribute('readonly')) continue;
            if (input.value.trim() !== '') return true;
        }
        
        const multiExclusiveInputs = cell.querySelectorAll<HTMLInputElement>('[data-multi-exclusive-value]');
        for (const input of multiExclusiveInputs) {
            if (input.value.trim() !== '') return true;
        }
    }
    return false;
}

function row_cells_clear_errors(cells: HTMLElement[]): void {
    cells.forEach(cell => {
        cell.querySelectorAll<HTMLInputElement>('input').forEach(input => {
            input_clear_error(input);
        });
    });
}

function row_cells_validate(cells: HTMLElement[]): boolean {
    if (!row_cells_has_data(cells)) {
        row_cells_clear_errors(cells);
        return true;
    }
    
    let valid = true;
    
    cells.forEach(cell => {
        cell.querySelectorAll<HTMLInputElement>('.number-input:not([readonly])').forEach(input => {
            const required = input.dataset.required === 'true';
            const error = validate_number_required(input);
            if (error) {
                input_show_error(input, error, required ? 'error' : 'warning');
                valid = false;
            } else {
                input_clear_error(input);
            }
        });
        
        cell.querySelectorAll<HTMLInputElement>('.string-input:not([readonly])').forEach(input => {
            const required = input.dataset.required === 'true';
            const error = validate_string_required(input);
            if (error) {
                input_show_error(input, error, required ? 'error' : 'warning');
                valid = false;
            } else {
                input_clear_error(input);
            }
        });
        
        cell.querySelectorAll<HTMLElement>('[data-enum-container]').forEach(container => {
            const input = container.querySelector('[data-enum-input]') as HTMLInputElement;
            if (input?.hasAttribute('readonly')) return;
            
            const required = input?.dataset.required === 'true';
            const error = validate_enum_required(container);
            if (error && input) {
                input_show_error(input, error, required ? 'error' : 'warning');
                valid = false;
            } else if (input) {
                input_clear_error(input);
            }
        });
        
        cell.querySelectorAll<HTMLElement>('[data-multi-exclusive-container]').forEach(container => {
            const hiddenInput = container.querySelector<HTMLInputElement>('[data-multi-exclusive-value]');
            const required = hiddenInput?.dataset.required === 'true';
            const error = validate_multi_exclusive_required(container);
            if (error && hiddenInput) {
                input_show_error(hiddenInput, error, required ? 'error' : 'warning');
                valid = false;
            } else if (hiddenInput) {
                input_clear_error(hiddenInput);
            }
        });
    }); 
    
    return valid;
}

function row_cells_show_success(cells: HTMLElement[]): void {
    cells.forEach(cell => {
        cell.querySelectorAll<HTMLInputElement>('.number-input:not([readonly])').forEach(input => {
            if (input.value.trim() !== '') input_show_success(input);
        });
        
        cell.querySelectorAll<HTMLInputElement>('.string-input:not([readonly])').forEach(input => {
            if (input.value.trim() !== '') input_show_success(input);
        });
        
        cell.querySelectorAll<HTMLElement>('[data-enum-container]').forEach(container => {
            const input = container.querySelector('[data-enum-input]') as HTMLInputElement;
            if (input?.hasAttribute('readonly')) return;
            const hidden = container.querySelector('[data-enum-value]') as HTMLInputElement;
            if (hidden?.value && input) input_show_success(input);
        });
    });
}



// ============================================================================
// Table Types
// ============================================================================

type TableType = 
    | 'HORIZONTAL_STATIC_UNIQUE'
    | 'HORIZONTAL_DYNAMIC_UNIQUE'
    | 'HORIZONTAL_DYNAMIC_DUPLICABLE'
    | 'VERTICAL_STATIC_UNIQUE'
    | 'SYSTEM_DEFINITION';

type StateTable = {
    element: HTMLElement;
    type: TableType;
    endpoint: string;
    enum_selected_index: Map<HTMLElement, number>;
    pending_save: boolean;
    last_save_time: number;
    // Dynamic table fields (only used for HORIZONTAL_DYNAMIC_*)
    is_dynamic: boolean;
    is_unique: boolean;
    row_counter: number;
};

const SAVE_COOLDOWN_MS = 3000;

// ============================================================================
// Table: Input Validation (single input, ignores required)
// ============================================================================

async function dynamic_table_load_existing(state: StateTable): Promise<void> {
    const jsonData = state.element.dataset.initial;
    if (!jsonData) return;

    let dataArray: Record<string, unknown>[];
    try {
        dataArray = JSON.parse(jsonData);
    } catch {
        console.error('Failed to parse initial data');
        return;
    }

    if (!Array.isArray(dataArray) || dataArray.length === 0) return;

    const firstRow = dataArray[0];
    if (!firstRow) return;

    const kodKey = Object.keys(firstRow).find(k => k.endsWith('_Kod'));
    if (!kodKey) return;

    for (const rowData of dataArray) {
        const code = rowData[kodKey] as string;
        if (!code) continue;

        const success = await dynamic_table_add_row(state, code);
        if (!success) continue;

        const rowIndex = state.row_counter - 1;
        const cells = row_cells_get(state.element, rowIndex);

        populate_cells_from_data(cells, rowData);
    }
}

function populate_cells_from_data(cells: HTMLElement[], rowData: Record<string, unknown>): void {
    cells.forEach(cell => {
        cell.querySelectorAll<HTMLInputElement>('.number-input').forEach(input => {
            const value = rowData[input.name];
            if (value !== undefined && value !== null) {
                const fmt = number_format_parse(input.dataset.format || '###0');
                input.value = number_format_value(Number(value), fmt);
            }
        });

        cell.querySelectorAll<HTMLInputElement>('.string-input').forEach(input => {
            const value = rowData[input.name];
            if (value !== undefined && value !== null) {
                input.value = String(value);
            }
        });

        cell.querySelectorAll<HTMLInputElement>('[data-enum-value]').forEach(hidden => {
            const value = rowData[hidden.name];
            if (value !== undefined && value !== null) {
                hidden.value = String(value);
                const container = hidden.closest('[data-enum-container]');
                const visibleInput = container?.querySelector('[data-enum-input]') as HTMLInputElement;
                const option = container?.querySelector(`[data-enum-option][data-value="${value}"]`) as HTMLElement;
                if (visibleInput && option) {
                    visibleInput.value = `${option.dataset.value} - ${option.dataset.label}`;
                }
            }
        });

        cell.querySelectorAll<HTMLInputElement>('[data-multi-exclusive-value]').forEach(hidden => {
            const value = rowData[hidden.name];
            if (value !== undefined && value !== null) {
                hidden.value = String(value);
                const container = hidden.closest('[data-multi-exclusive-container]') as HTMLElement;
                if (container) multi_exclusive_load_value(container);
            }
        });
    });
}

async function dynamic_table_add_row(state: StateTable, code: string): Promise<boolean> {
    const index = state.row_counter++;
    const url = `${state.endpoint.replace(/\/$/, '')}/${code}/${index}`;
    
    try {
        const response = await fetch(url);
        if (!response.ok) throw new Error(`${response.status}`);
        
        const html = await response.text();
        state.element.insertAdjacentHTML('beforeend', html);
        
        if (state.is_unique) {
            const option = state.element.querySelector(
                `[data-row-selector] [data-enum-option][data-value="${code}"]`
            ) as HTMLElement;
            option?.classList.add('hidden');
        }
        
        zebra_striping_apply(state.element);
        
        const input = state.element.querySelector('[data-row-selector] [data-enum-input]') as HTMLInputElement;
        if (input) input.value = '';
        
        state.element.querySelector('[data-row-selector] [data-enum-dropdown]')?.classList.add('hidden');
        
        return true;
    } catch (err) {
        toast_show(`Błąd dodawania wiersza: ${err}`, 'error');
        return false;
    }
}

function dynamic_table_delete_row(state: StateTable, cell: HTMLElement): void {
    const rowIndex = cell.dataset.rowIndex;
    const rowCode = cell.dataset.rowCode;
    if (!rowIndex || !rowCode) return;
    
    state.element.querySelectorAll(
        `[data-cell][data-row-index="${rowIndex}"]`
    ).forEach(el => el.remove());
    
    // Unhide option in unique mode
    if (state.is_unique) {
        const option = state.element.querySelector(
            `[data-row-selector] [data-enum-option][data-row-code="${rowCode}"]`
        ) as HTMLElement;
        option?.classList.remove('hidden');
    }
    
    zebra_striping_apply(state.element);
}

function dynamic_table_handle_selector_input(state: StateTable, target: HTMLInputElement): void {
    const container = target.closest('[data-enum-container]') as HTMLElement;
    const dropdown = container.querySelector('[data-enum-dropdown]') as HTMLElement;
    const query = target.value.toLowerCase();
    
    dropdown?.classList.remove('hidden');
    state.enum_selected_index.set(container, -1);
    
    container.querySelectorAll('[data-enum-option]').forEach(opt => {
        const el = opt as HTMLElement;
        if (state.is_unique && el.classList.contains('hidden')) return; // Already hidden by unique logic
        
        const text = `${el.dataset.value} - ${el.dataset.label}`.toLowerCase();
        el.classList.toggle('hidden', !text.includes(query));
        el.classList.remove('bg-blue-100');
    });
}

function dynamic_table_handle_selector_keydown(state: StateTable, e: KeyboardEvent, target: HTMLInputElement): void {
    const container = target.closest('[data-enum-container]') as HTMLElement;
    const dropdown = container.querySelector('[data-enum-dropdown]') as HTMLElement;
    const is_visible = dropdown && !dropdown.classList.contains('hidden');
    
    if (e.key === 'Escape') {
        e.preventDefault();
        dropdown?.classList.add('hidden');
        return;
    }
    
    if (e.key === 'ArrowDown' && !is_visible) {
        e.preventDefault();
        dropdown?.classList.remove('hidden');
        return;
    }
    
    if (!is_visible) return;
    
    const options = Array.from(container.querySelectorAll('[data-enum-option]:not(.hidden)')) as HTMLElement[];
    let index = state.enum_selected_index.get(container) ?? -1;
    
    if (e.key === 'ArrowDown') {
        e.preventDefault();
        index = Math.min(index + 1, options.length - 1);
    } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        index = Math.max(index - 1, 0);
    } else if (e.key === 'Enter') {
        e.preventDefault();
        const selected = options[index];
        if (index >= 0 && selected?.dataset.rowCode) {
            dynamic_table_add_row(state, selected.dataset.rowCode);
        }
        return;
    }
    
    state.enum_selected_index.set(container, index);
    options.forEach((opt, i) => opt.classList.toggle('bg-blue-100', i === index));
    options[index]?.scrollIntoView({ block: 'nearest' });
}

function validate_number_value(input: HTMLInputElement): string | null {
    const raw = input.value.trim();
    if (string_is_blank(raw)) return null;
    
    const value = number_value_parse(raw);
    const min = input.dataset.min ? parseFloat(input.dataset.min) : null;
    const max = input.dataset.max ? parseFloat(input.dataset.max) : null;
    const fmt = number_format_parse(input.dataset.format || '###0');
    
    if (value === null) {
        return 'Nieprawidłowy format liczby';
    }
    
    if (min !== null && value < min) {
        return `Wartość musi być co najmniej ${number_format_value(min, fmt)}`;
    }
    
    if (max !== null && value > max) {
        return `Wartość musi być co najwyżej ${number_format_value(max, fmt)}`;
    }
    
    return null;
}

function validate_number_required(input: HTMLInputElement): string | null {
    const raw = input.value.trim();
    const required = input.dataset.required === 'true';
    const custom_msg = input.dataset.errorMessage;
    
    if (required && string_is_blank(raw)) {
        return custom_msg || 'To pole jest wymagane';
    }
    
    return validate_number_value(input);
}

function validate_string_required(input: HTMLInputElement): string | null {
    const required = input.dataset.required === 'true';
    const value = input.value.trim();
    
    if (required && !value) {
        return 'To pole jest wymagane';
    }
    
    return input_format_error_get(input);
}

function validate_enum_required(container: HTMLElement): string | null {
    const input = container.querySelector('[data-enum-input]') as HTMLInputElement;
    const hidden = container.querySelector('[data-enum-value]') as HTMLInputElement;
    const required = input?.dataset.required === 'true';
    
    if (required && !hidden?.value) {
        return 'Wybierz wartość z listy';
    }
    
    return null;
}

// ============================================================================
// String Input Validation
// ============================================================================

function validate_string_pattern(value: string, format: string): boolean {
    if (format === '$') return true;
    
    const patterns = [
        /^\d{1}$/,
        /^\d{2}$/,
        /^\d{2}-\d{2}$/,
        /^\d{2}-\d{2}-\d{2}$/,
        /^\d{2}-\d{2}-\d{2}-\d{2}$/
    ];
    
    return value === '' || patterns.some(pattern => pattern.test(value));
}

function string_format_pattern(value: string): string {
    const digits = value.replace(/[^0-9]/g, '');
    let formatted = '';
    
    for (let i = 0; i < digits.length && i < 8; i++) {
        if (i > 0 && i % 2 === 0) {
            formatted += '-';
        }
        formatted += digits[i];
    }
    
    return formatted;
}

function input_format_error_get(input: HTMLInputElement): string | null {
    const format = input.dataset.format;
    const value = input.value;
    
    if (!format || format === '$') return null;
    
    if (format.includes('#')) {
        if (value && !validate_string_pattern(value, format)) {
            return 'Nieprawidłowy format (np. 12, 12-34, 12-34-56)';
        }
    }
    
    return null;
}

// ============================================================================
// Table Validation & Serialization
// ============================================================================

function table_validate_all(state: StateTable): boolean {
    let valid = true;
    
    for (const rowIndex of all_row_indices_get(state)) {
        const cells = row_cells_get(state.element, rowIndex);
        if (!row_cells_validate(cells)) {
            valid = false;
        }
    }
    
    return valid;
}

function table_serialize(state: StateTable): Record<string, unknown>[] {
    const rows: Record<string, unknown>[] = [];
    
    for (const rowIndex of all_row_indices_get(state)) {
        const cells = row_cells_get(state.element, rowIndex);
        if (!row_cells_has_data(cells)) continue;
        
        const data: Record<string, unknown> = {};
        
        cells.forEach(c => {
            c.querySelectorAll<HTMLInputElement>('.number-input').forEach(input => {
                const value = number_value_parse(input.value);
                if (value !== null) data[input.name] = value;
            });
            c.querySelectorAll<HTMLInputElement>('.string-input').forEach(input => {
                const value = input.value.trim();
                if (value) data[input.name] = value;
            });
            c.querySelectorAll<HTMLInputElement>('[data-enum-value]').forEach(input => {
                if (input.value) data[input.name] = input.value;
            });
            c.querySelectorAll<HTMLInputElement>('[data-multi-exclusive-value]').forEach(input => {
                if (input.value) data[input.name] = input.value;
            });
        });
        
        rows.push(data);
    }
    
    return rows;
}

function table_serialize_vertical(state: StateTable): Record<string, unknown> {
    const data: Record<string, unknown> = {};
    
    state.element.querySelectorAll<HTMLInputElement>('.number-input').forEach(input => {
        const value = number_value_parse(input.value);
        if (value !== null) data[input.name] = value;
    });
    
    state.element.querySelectorAll<HTMLInputElement>('.string-input').forEach(input => {
        const value = input.value.trim();
        if (value) data[input.name] = value;
    });
    
    state.element.querySelectorAll<HTMLInputElement>('[data-enum-value]').forEach(input => {
        if (input.value) data[input.name] = input.value;
    });
    
    state.element.querySelectorAll<HTMLInputElement>('[data-multi-exclusive-value]').forEach(input => {
        if (input.value) data[input.name] = input.value;
    });
    
    return data;
}

// ============================================================================
// Table: Save
// ============================================================================

async function table_save(state: StateTable): Promise<boolean> {
    if (state.pending_save) return false;
    
    const now = Date.now();
    const time_since_last = now - state.last_save_time;
    if (time_since_last < SAVE_COOLDOWN_MS) {
        const remaining = Math.ceil((SAVE_COOLDOWN_MS - time_since_last) / 1000);
        toast_show(`Poczekaj ${remaining}s przed kolejnym zapisem`, 'warning');
        return false;
    }
    
    if (!table_validate_all(state)) {
        toast_show('Formularz zawiera błędy', 'error');
        return false;
    }
    
    let data: Record<string, unknown> | Record<string, unknown>[];
    
    if (state.type === 'VERTICAL_STATIC_UNIQUE') {
        data = table_serialize_vertical(state);
    } else {
        data = table_serialize(state);
    }
    
    if (state.type === 'VERTICAL_STATIC_UNIQUE') {
        if (Object.keys(data).length === 0) {
            toast_show('Brak danych do zapisania', 'warning');
            return false;
        }
    } else {
        if ((data as Record<string, unknown>[]).length === 0) {
            toast_show('Brak danych do zapisania', 'warning');
            return false;
        }
    }
    
    state.pending_save = true;
    
    try {
        const response = await fetch(state.endpoint, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data),
        });
        
        if (!response.ok) {
            throw new Error(`Błąd serwera: ${response.status}`);
        }
        
        state.last_save_time = Date.now();
        
        for (const rowIndex of all_row_indices_get(state)) {
            const cells = row_cells_get(state.element, rowIndex);
            if (row_cells_has_data(cells)) {
                row_cells_show_success(cells);
            }
        }
        
        toast_show('Zapisano pomyślnie', 'success');
        return true;
        
    } catch (err) {
        toast_show(`Błąd zapisu: ${err}`, 'error');
        return false;
        
    } finally {
        state.pending_save = false;
    }
}

// ============================================================================
// Table: Event Handlers
// ============================================================================

function table_handle_input_number(target: HTMLInputElement): void {
    const format_str = target.dataset.format ?? '###0';
    const fmt = number_format_parse(format_str);
    
    let value = target.value;
    
    value = value.replace('.', ',');
    
    if (fmt.decimals > 0) {
        value = value.replace(/[^\d,\-]/g, '');
    } else {
        value = value.replace(/[^\d\-]/g, '');
    }
    
    if (value.includes('-')) {
        const first_char = value.charAt(0) === '-' ? '-' : '';
        value = first_char + value.replace(/-/g, '');
    }
    
    const comma_count = (value.match(/,/g) || []).length;
    if (comma_count > 1) {
        const first_comma = value.indexOf(',');
        value = value.substring(0, first_comma + 1) + value.substring(first_comma + 1).replace(/,/g, '');
    }
    
    if (fmt.decimals > 0 && value.includes(',')) {
        const [int_part = '', dec_part = ''] = value.split(',');
        value = int_part + ',' + dec_part.substring(0, fmt.decimals);
    }
    
    target.value = value;
    
    const error = validate_number_value(target);
    if (error) {
        const required = target.dataset.required === 'true';
        input_show_error(target, error, required ? 'error' : 'warning');
        input_show_error_popup(target);
    } else {
        input_clear_error(target);
    }
}

function table_handle_input_string(target: HTMLInputElement): void {
    const format = target.dataset.format ?? '$';
    if (format === '$') {
        return;
    }
    
    if (format.includes('#')) {
        target.value = string_format_pattern(target.value);
    }
    
    const error = input_format_error_get(target);
    if (error) {
        const required = target.dataset.required === 'true';
        input_show_error(target, error, required ? 'error' : 'warning');
        input_show_error_popup(target);
    } else {
        input_clear_error(target);
    }
}

function enum_dropdown_position(input: HTMLInputElement, dropdown: HTMLElement): void {
    const inputRect = input.getBoundingClientRect();
    
    // Calculate if input is past 70% of viewport height
    const threshold = window.innerHeight * 0.7;
    const showAbove = inputRect.bottom > threshold;
    
    dropdown.style.position = 'fixed';
    dropdown.style.left = `${inputRect.left}px`;
    dropdown.style.width = `${inputRect.width}px`;
    
    if (showAbove) {
        dropdown.style.bottom = `${window.innerHeight - inputRect.top}px`;
        dropdown.style.top = 'auto';
    } else {
        dropdown.style.top = `${inputRect.bottom}px`;
        dropdown.style.bottom = 'auto';
    }
}

function enum_dropdown_filter(state: StateTable, target: HTMLInputElement): void {
    const container = target.closest('[data-enum-container]') as HTMLElement;
    const dropdown = container.querySelector('[data-enum-dropdown]') as HTMLElement;
    const query = target.value.toLowerCase();
    
    dropdown?.classList.remove('hidden');
    state.enum_selected_index.set(container, -1);
    
    enum_dropdown_position(target, dropdown);
    
    container.querySelectorAll('[data-enum-option]').forEach(opt => {
        const el = opt as HTMLElement;
        const text = `${el.dataset.value} - ${el.dataset.label}`.toLowerCase();
        el.classList.toggle('hidden', !text.includes(query));
        el.classList.remove('bg-blue-100');
    });
}

function input_event_route(state: StateTable, e: Event): void {
    const target = e.target as HTMLInputElement;
    
    if (state.is_dynamic && target.hasAttribute('data-row-adder')) {
        dynamic_table_handle_selector_input(state, target);
        return;
    }

    if (target.classList.contains('number-input')) {
        table_handle_input_number(target);
        return;
    }

    if (target.classList.contains('string-input')) {
        table_handle_input_string(target);
        return;
    }

    if (target.hasAttribute('data-enum-input')) {
        enum_dropdown_filter(state, target);
        return;
    }
}

function table_handle_focus(state: StateTable, e: Event): void {
    const target = e.target as HTMLElement;
    
    if (target instanceof HTMLInputElement && target.dataset.errorMessage) {
        state.element.querySelectorAll('.input-error-popup').forEach(p => p.remove());
        input_show_error_popup(target);
    }
    
    if (target.hasAttribute('data-enum-input')) {
        state.element.querySelectorAll('[data-enum-dropdown]').forEach(d => {
            d.classList.add('hidden');
        });
        
        const container = target.closest('[data-enum-container]') as HTMLElement;
        const dropdown = container.querySelector('[data-enum-dropdown]') as HTMLElement;
        
        if (dropdown) {
            enum_dropdown_position(target as HTMLInputElement, dropdown);
            dropdown.classList.remove('hidden');
        }
        state.enum_selected_index.set(container, -1);
    }
}

function table_handle_blur(state: StateTable, e: Event): void {
    const target = e.target as HTMLInputElement;
    const cell = target.closest('[data-cell]') as HTMLElement;
    if (!cell) return;
    
    cell.querySelector('.input-error-popup')?.remove();
    
    if (target.classList.contains('number-input')) {
        const format_str = target.dataset.format || '###0';
        const fmt = number_format_parse(format_str);
        const value = number_value_parse(target.value);
        
        if (value !== null) {
            target.value = number_format_value(value, fmt);
        }
    }
    
    const rowIndex = row_index_get(cell);
    if (rowIndex === null) return;
    
    setTimeout(() => {
        const cells = row_cells_get(state.element, rowIndex);
        if (!row_cells_has_data(cells)) {
            row_cells_clear_errors(cells);
        }
    }, 0);
}

function table_handle_keydown(state: StateTable, e: KeyboardEvent): void {
    const target = e.target as HTMLInputElement;
    
    if (state.is_dynamic && target.hasAttribute('data-row-adder')) {
        dynamic_table_handle_selector_keydown(state, e, target);
        return;
    }
    
    if (target.classList.contains('string-input')) {
        const format = target.dataset.format;
        if (format && format.includes('#')) {
            const allowed = ['Backspace', 'Delete', 'ArrowLeft', 'ArrowRight', 'Tab', 'Enter'];
            if (!allowed.includes(e.key) && !/[0-9]/.test(e.key)) {
                e.preventDefault();
                return;
            }
        }
    }
    
    if (e.key === 'Enter' && !target.hasAttribute('data-enum-input')) {
        e.preventDefault();
        
        const cell = target.closest('[data-cell]') as HTMLElement;
        const rowIndex = cell ? row_index_get(cell) : null;
        
        if (rowIndex !== null) {
            const cells = row_cells_get(state.element, rowIndex);
            if (row_cells_validate(cells)) {
                table_save(state);
            } else {
                toast_show('Uzupełnij wymagane pola w tym wierszu', 'error');
            }
        }
        return;
    }
    
    if (target.hasAttribute('data-enum-input')) {
        const container = target.closest('[data-enum-container]') as HTMLElement;
        const dropdown = container.querySelector('[data-enum-dropdown]') as HTMLElement;
        const is_visible = dropdown && !dropdown.classList.contains('hidden');
        
        if (e.key === 'Escape') {
            e.preventDefault();
            dropdown?.classList.add('hidden');
            return;
        }
        
        if (e.key === 'ArrowDown' && !is_visible) {
            e.preventDefault();
            dropdown?.classList.remove('hidden');
            return;
        }
        
        if (!is_visible) return;
        
        const options = Array.from(container.querySelectorAll('[data-enum-option]:not(.hidden)')) as HTMLElement[];
        let index = state.enum_selected_index.get(container) ?? -1;
        
        if (e.key === 'ArrowDown') {
            e.preventDefault();
            index = Math.min(index + 1, options.length - 1);
        } else if (e.key === 'ArrowUp') {
            e.preventDefault();
            index = Math.max(index - 1, 0);
        } else if (e.key === 'Enter') {
            e.preventDefault();
            const selected = options[index];
            if (index >= 0 && selected) {
                table_enum_select(selected);
                
                const cell = target.closest('[data-cell]') as HTMLElement;
                const rowIndex = cell ? row_index_get(cell) : null;
                
                if (rowIndex !== null) {
                    const cells = row_cells_get(state.element, rowIndex);
                    if (row_cells_validate(cells)) {
                        table_save(state);
                    } else {
                        toast_show('Uzupełnij wymagane pola w tym wierszu', 'error');
                    }
                }
            }
            return;
        }
        
        state.enum_selected_index.set(container, index);
        options.forEach((opt, i) => opt.classList.toggle('bg-blue-100', i === index));
        
        const current = options[index];
        current?.scrollIntoView({ block: 'nearest' });
    }
}

function table_handle_click(state: StateTable, e: Event): void {
    const target = e.target as HTMLElement;
    
    // Handle enum input click (for when already focused)
    if (target.hasAttribute('data-enum-input')) {
        state.element.querySelectorAll('[data-enum-dropdown]').forEach(d => {
            d.classList.add('hidden');
        });
        
        const container = target.closest('[data-enum-container]') as HTMLElement;
        const dropdown = container.querySelector('[data-enum-dropdown]') as HTMLElement;
        
        if (dropdown) {
            enum_dropdown_position(target as HTMLInputElement, dropdown);
            dropdown.classList.remove('hidden');
        }
        state.enum_selected_index.set(container, -1);
        return;
    }
    
    // Dynamic table: row selector option
    if (state.is_dynamic) {
        const selectorOption = target.closest('[data-row-selector] [data-enum-option]') as HTMLElement;
        if (selectorOption?.dataset.rowCode) {
            e.preventDefault();
            dynamic_table_add_row(state, selectorOption.dataset.rowCode);
            return;
        }
        
        const deleteBtn = target.closest('[data-delete-row]') as HTMLElement;
        if (deleteBtn) {
            e.preventDefault();
            const cell = deleteBtn.closest('[data-cell]') as HTMLElement;
            if (cell) dynamic_table_delete_row(state, cell);
            return;
        }
    } 
    
    // Regular enum option
    const option = target.closest('[data-enum-option]') as HTMLElement;
    if (option) {
        e.preventDefault();
        table_enum_select(option);
        
        const cell = option.closest('[data-cell]') as HTMLElement;
        const rowIndex = cell ? row_index_get(cell) : null;
        
        if (rowIndex !== null) {
            const cells = row_cells_get(state.element, rowIndex);
            if (row_cells_validate(cells)) {
                table_save(state);
            } else {
                toast_show('Uzupełnij wymagane pola w tym wierszu', 'error');
            }
        }
        return;
    }
}

function table_handle_mousedown(e: Event): void {
    const target = e.target as HTMLElement;
    
    if (target.closest('[data-enum-option]')) {
        e.preventDefault();
    }
}

// ============================================================================
// Table: Enum Selection
// ============================================================================

function table_enum_select(option: HTMLElement): void {
    const container = option.closest('[data-enum-container]') as HTMLElement;
    const input = container.querySelector('[data-enum-input]') as HTMLInputElement;
    const hidden = container.querySelector('[data-enum-value]') as HTMLInputElement;
    const dropdown = container.querySelector('[data-enum-dropdown]') as HTMLElement;
    
    const value = option.dataset.value || '';
    const label = option.dataset.label || '';
    
    input.value = `${value} - ${label}`;
    hidden.value = value;
    dropdown.classList.add('hidden');
    
    input_clear_error(input);
}

function enum_select_init(state: StateTable): void {        
    state.element.querySelectorAll<HTMLElement>('[data-enum-container]').forEach(element => {
        if (element.hasAttribute('data-enum-row-selector')) return
        
        const input = element.querySelector('[data-enum-input]') as HTMLInputElement;
        const hidden = element.querySelector('[data-enum-value]') as HTMLInputElement;
        const dropdown = element.querySelector('[data-enum-dropdown]') as HTMLElement;
    
        const chosen = hidden.value;
        for (const option of dropdown.querySelectorAll('[data-enum-option]')) {
            const text = option.textContent;
            if (text.includes(chosen)) {
                input.value = text.trim();
                break
            }
        }
    })
}

// ============================================================================
// Zebra Striping
// ============================================================================

function zebra_striping_apply(table: HTMLElement): void {
    table.querySelectorAll<HTMLElement>('[data-cell][data-row-index]').forEach(cell => {
        const index = parseInt(cell.dataset.rowIndex ?? '0', 10);
        cell.classList.toggle('bg-white', index % 2 === 0);
        cell.classList.toggle('bg-slate-100/70', index % 2 === 1);
    });
}

// ============================================================================
// Table: Initialization
// ============================================================================

function table_init(element: HTMLElement): StateTable {
    const tableType = (element.dataset.tableType as TableType) || 'HORIZONTAL_STATIC_UNIQUE';
    const is_dynamic = tableType === 'HORIZONTAL_DYNAMIC_UNIQUE' || tableType === 'HORIZONTAL_DYNAMIC_DUPLICABLE';
    
    const state: StateTable = {
        element,
        type: tableType,
        endpoint: element.dataset.endpoint!,
        enum_selected_index: new Map(),
        pending_save: false,
        last_save_time: 0,
        is_dynamic,
        is_unique: tableType === 'HORIZONTAL_DYNAMIC_UNIQUE',
        row_counter: 0,
    };
    
    zebra_striping_apply(element)
    multi_exclusive_init(state)
    enum_select_init(state)
     
    if (state.is_dynamic) {
        dynamic_table_load_existing(state);
    }
    
    element.addEventListener('input', (e) => input_event_route(state, e));
    element.addEventListener('focus', (e) => table_handle_focus(state, e), true);
    element.addEventListener('blur', (e) => table_handle_blur(state, e), true);
    element.addEventListener('keydown', (e) => table_handle_keydown(state, e));
    element.addEventListener('click', (e) => table_handle_click(state, e));
    element.addEventListener('mousedown', (e) => table_handle_mousedown(e));
    
    return state;
}

// ============================================================================
// Tooltips
// ============================================================================

function tooltips_init(): void {
    const tooltip = document.createElement('div'); 
    tooltip.className = `
        fixed px-3 py-2 font-medium text-white
        bg-slate-800/95 backdrop-blur-sm rounded-lg
        shadow-[0_4px_20px_rgba(0,0,0,0.25)]
        pointer-events-none opacity-0 transition-opacity duration-150 z-[9999]
        max-w-[200px] whitespace-normal break-words
    `.trim().replace(/\s+/g, ' ');

    document.body.appendChild(tooltip);

    document.addEventListener('mouseenter', (e) => {
        if (!(e.target instanceof Element)) return;
        const target = (e.target as HTMLElement).closest<HTMLElement>('[data-tooltip]');
        if (!target || !target.dataset.tooltip) return;

        tooltip.textContent = target.dataset.tooltip;
        tooltip.classList.remove('opacity-0');
        tooltip.classList.add('opacity-100');

        const rect = target.getBoundingClientRect();
        tooltip.style.left = `${rect.left + rect.width / 2 - tooltip.offsetWidth / 2}px`;
        tooltip.style.top = `${rect.bottom + 8}px`;
    }, true);

    document.addEventListener('mouseleave', (e) => {
        if (!(e.target instanceof Element)) return;
        const target = (e.target as HTMLElement).closest<HTMLElement>('[data-tooltip]');
        if (!target) return;

        tooltip.classList.remove('opacity-100');
        tooltip.classList.add('opacity-0');
    }, true);
}

// ============================================================================
// Multi-Exclusive Input
// ============================================================================

function multi_exclusive_load_value(container: HTMLElement): void {
    const exclusiveCheckbox = container.querySelector<HTMLInputElement>('[data-exclusive-option]');
    const regularCheckboxes = container.querySelectorAll<HTMLInputElement>('[data-regular-option]');
    const hiddenInput = container.querySelector<HTMLInputElement>('[data-multi-exclusive-value]');

    if (!hiddenInput?.value) return;

    const selectedValues = hiddenInput.value.split(',').map(v => v.trim()).filter(Boolean);
    
    if (exclusiveCheckbox && selectedValues.includes(exclusiveCheckbox.value)) {
        exclusiveCheckbox.checked = true;
        regularCheckboxes.forEach(cb => (cb.checked = false));
        return
    } 
    
    if (exclusiveCheckbox) exclusiveCheckbox.checked = false;
    regularCheckboxes.forEach(cb => {
        cb.checked = selectedValues.includes(cb.value);
    });
}

function multi_exclusive_update_value(container: HTMLElement): void {
    const exclusiveCheckbox = container.querySelector<HTMLInputElement>('[data-exclusive-option]');
    const regularCheckboxes = container.querySelectorAll<HTMLInputElement>('[data-regular-option]');
    const hiddenInput = container.querySelector<HTMLInputElement>('[data-multi-exclusive-value]');

    if (!hiddenInput) return;

    const selected: string[] = [];

    if (exclusiveCheckbox?.checked) {
        selected.push(exclusiveCheckbox.value);
    } else {
        regularCheckboxes.forEach(cb => {
            if (cb.checked) selected.push(cb.value);
        });
    }

    hiddenInput.value = selected.join(',');
}

function multi_exclusive_handle_change(container: HTMLElement, target: HTMLInputElement, state: StateTable): void {
    const exclusiveCheckbox = container.querySelector<HTMLInputElement>('[data-exclusive-option]');
    const regularCheckboxes = container.querySelectorAll<HTMLInputElement>('[data-regular-option]');

    if (target.hasAttribute('data-exclusive-option')) {
        if (target.checked) {
            regularCheckboxes.forEach(cb => (cb.checked = false));
        }
    } else {
        if (target.checked && exclusiveCheckbox) {
            exclusiveCheckbox.checked = false;
        }
    }

    multi_exclusive_update_value(container);
    
    const cell = container.closest('[data-cell]') as HTMLElement;
    const rowIndex = cell ? row_index_get(cell) : null;
    
    
    if (rowIndex !== null) {
        const cells = row_cells_get(state.element, rowIndex);
        if (row_cells_validate(cells)) {
            table_save(state);
        }
    }
}

function validate_multi_exclusive_required(container: HTMLElement): string | null {
    const hiddenInput = container.querySelector<HTMLInputElement>('[data-multi-exclusive-value]');
    const required = hiddenInput?.dataset.required === 'true';
    
    if (required && !hiddenInput?.value) {
        return 'Wybierz co najmniej jedną opcję';
    }
    
    return null;
}

function multi_exclusive_init(state: StateTable): void {
    state.element.querySelectorAll<HTMLElement>('[data-multi-exclusive-container]').forEach(container => {
        multi_exclusive_load_value(container);

        container.addEventListener('change', (e) => {
            const target = e.target as HTMLInputElement;
            if (target.hasAttribute('data-multi-option')) {
                multi_exclusive_handle_change(container, target, state);
            }
        });
    });
}

// ============================================================================
// Table Statusy 
// ============================================================================

type StateTableStatusy = {
    element: HTMLElement;
}

function table_statusy_init(element: HTMLElement): StateTableStatusy {
    const state: StateTableStatusy = {
        element,
    };

    table_statusy_zebra_apply(element);
    
    element.addEventListener('click', (e) => table_select_handle_click(e));

    return state;
}

function table_statusy_zebra_apply(table: HTMLElement): void {
    table.querySelectorAll<HTMLElement>('tr[data-row-index]').forEach(row => {
        const index = parseInt(row.dataset.rowIndex ?? '0', 10);
        row.classList.toggle('bg-white', index % 2 === 0);
        row.classList.toggle('bg-slate-50', index % 2 === 1);
    });
}

function table_select_handle_click(e: Event): void {
    const target = e.target as HTMLElement;
    const row = target.closest<HTMLElement>('tr[data-row-url]');
    
    if (row && row.dataset.rowUrl) {
        window.location.href = row.dataset.rowUrl;
    }
}

// ============================================================================
// Main
// ============================================================================

document.addEventListener('DOMContentLoaded', () => {
    document_click_outside_init();
    
    year_select_init();
    user_menu_init();
    logout_init();
    session_timer_init(30);
    nav_toggle_init();    
    tooltips_init();

    document.querySelectorAll<HTMLElement>('[data-table-type]').forEach(table_init);
    document.querySelectorAll<HTMLElement>('[data-table-statusy]').forEach(table_statusy_init);
});