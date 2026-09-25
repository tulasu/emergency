import { Component, input, model } from '@angular/core';

@Component({
  selector: 'label[tbCheckRow]',
  template: `
    <span class="tb-check" [class.tb-check--on]="checked()">
      @if (checked()) {
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M5 13l4 4L19 7" fill="none" stroke="currentColor" stroke-width="2" />
        </svg>
      }
    </span>
    <span>{{ label() }}</span>
    <input type="checkbox" hidden [checked]="checked()" (change)="checked.set($any($event.target).checked)" />
  `,
  styleUrl: './check-row.css',
})
export class TbCheckRow {
  readonly label = input.required<string>();
  readonly checked = model(true);
}
