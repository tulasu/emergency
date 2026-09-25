import { Component, ElementRef, forwardRef, HostListener, inject, input, signal } from '@angular/core';
import { ControlValueAccessor, NG_VALUE_ACCESSOR } from '@angular/forms';
import { TbIcon } from '../icon/icon';

export interface TbSelectOption {
  value: string;
  label: string;
}

@Component({
  selector: 'tb-select',
  imports: [TbIcon],
  template: `
    <button
      type="button"
      class="tb-select__btn"
      [class.tb-select__btn--open]="open()"
      [disabled]="disabled()"
      (click)="toggle($event)"
    >
      <span [class.tb-select__placeholder]="!selectedLabel()">{{ selectedLabel() || placeholder() }}</span>
      <tb-icon name="chevron-down" />
    </button>
    @if (open()) {
      <ul class="tb-select__list" role="listbox">
        @for (opt of options(); track opt.value) {
          <li>
            <button type="button" class="tb-select__opt" (click)="pick(opt, $event)">
              <span>{{ opt.label }}</span>
              @if (opt.value === value()) {
                <tb-icon name="check" />
              }
            </button>
          </li>
        }
      </ul>
    }
  `,
  styleUrl: './select.css',
  providers: [
    {
      provide: NG_VALUE_ACCESSOR,
      useExisting: forwardRef(() => TbSelect),
      multi: true,
    },
  ],
  host: {
    class: 'tb-select',
  },
})
export class TbSelect implements ControlValueAccessor {
  readonly options = input<TbSelectOption[]>([]);
  readonly placeholder = input('Выберите');

  readonly value = signal('');
  readonly disabled = signal(false);
  readonly open = signal(false);

  private readonly host = inject(ElementRef<HTMLElement>);
  private onChange: (value: string) => void = () => undefined;
  private onTouched: () => void = () => undefined;

  selectedLabel(): string {
    return this.options().find((opt) => opt.value === this.value())?.label ?? '';
  }

  writeValue(value: string | null): void {
    this.value.set(value ?? '');
  }

  registerOnChange(fn: (value: string) => void): void {
    this.onChange = fn;
  }

  registerOnTouched(fn: () => void): void {
    this.onTouched = fn;
  }

  setDisabledState(isDisabled: boolean): void {
    this.disabled.set(isDisabled);
  }

  toggle(event: Event): void {
    event.stopPropagation();
    if (this.disabled()) {
      return;
    }
    this.open.update((v) => !v);
    this.onTouched();
  }

  pick(opt: TbSelectOption, event: Event): void {
    event.stopPropagation();
    this.value.set(opt.value);
    this.onChange(opt.value);
    this.open.set(false);
    this.onTouched();
  }

  @HostListener('document:click', ['$event'])
  close(event: Event): void {
    if (!this.host.nativeElement.contains(event.target as Node)) {
      this.open.set(false);
    }
  }

  @HostListener('document:keydown.escape')
  closeOnEsc(): void {
    this.open.set(false);
  }
}
