import { Component, forwardRef, input, signal } from '@angular/core';
import { ControlValueAccessor, NG_VALUE_ACCESSOR } from '@angular/forms';

@Component({
  selector: 'tb-input',
  template: `
    <input
      class="tb-input"
      [type]="type()"
      [placeholder]="placeholder()"
      [attr.autocomplete]="autocomplete() || null"
      [disabled]="disabled()"
      [value]="value()"
      (input)="onInput($event)"
      (blur)="touched()"
    />
  `,
  styleUrl: './input.css',
  providers: [
    {
      provide: NG_VALUE_ACCESSOR,
      useExisting: forwardRef(() => TbInput),
      multi: true,
    },
  ],
})
export class TbInput implements ControlValueAccessor {
  readonly type = input('text');
  readonly placeholder = input('');
  readonly autocomplete = input('');

  readonly value = signal('');
  readonly disabled = signal(false);

  private onChange: (value: string) => void = () => undefined;
  private onTouched: () => void = () => undefined;

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

  onInput(event: Event): void {
    const next = (event.target as HTMLInputElement).value;
    this.value.set(next);
    this.onChange(next);
  }

  touched(): void {
    this.onTouched();
  }
}
