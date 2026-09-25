import { Component, input } from '@angular/core';

export type TbIconName =
  | 'arrow-left'
  | 'plus'
  | 'layers'
  | 'folder'
  | 'users'
  | 'user-plus'
  | 'settings'
  | 'log-out'
  | 'user'
  | 'copy'
  | 'chevron-down'
  | 'check'
  | 'x'
  | 'file';

@Component({
  selector: 'tb-icon',
  template: `
    @switch (name()) {
      @case ('arrow-left') {
        <svg viewBox="0 0 24 24"><path d="M15 18l-6-6 6-6" /></svg>
      }
      @case ('plus') {
        <svg viewBox="0 0 24 24"><path d="M12 5v14M5 12h14" /></svg>
      }
      @case ('layers') {
        <svg viewBox="0 0 24 24"><path d="M12 3l9 5-9 5-9-5 9-5zM3 12l9 5 9-5M3 16l9 5 9-5" /></svg>
      }
      @case ('folder') {
        <svg viewBox="0 0 24 24"><path d="M3 7h6l2 2h10v10H3z" /></svg>
      }
      @case ('users') {
        <svg viewBox="0 0 24 24"><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8zM22 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75" /></svg>
      }
      @case ('user-plus') {
        <svg viewBox="0 0 24 24">
          <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" />
          <circle cx="9" cy="7" r="4" />
          <path d="M19 8v6M22 11h-6" />
        </svg>
      }
      @case ('settings') {
        <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="3" /><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.8-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1-1.5 1.7 1.7 0 0 0-1.8.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.8 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.5-1 1.7 1.7 0 0 0-.3-1.8l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.8.3H9a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.8-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.8V9c0 .7.4 1.3 1.1 1.5H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1z" /></svg>
      }
      @case ('log-out') {
        <svg viewBox="0 0 24 24"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5M21 12H9" /></svg>
      }
      @case ('user') {
        <svg viewBox="0 0 24 24"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" /><circle cx="12" cy="7" r="4" /></svg>
      }
      @case ('copy') {
        <svg viewBox="0 0 24 24"><rect x="9" y="9" width="13" height="13" rx="2" /><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" /></svg>
      }
      @case ('chevron-down') {
        <svg viewBox="0 0 24 24"><path d="m6 9 6 6 6-6" /></svg>
      }
      @case ('check') {
        <svg viewBox="0 0 24 24"><path d="M20 6 9 17l-5-5" /></svg>
      }
      @case ('x') {
        <svg viewBox="0 0 24 24"><path d="M18 6 6 18M6 6l12 12" /></svg>
      }
      @case ('file') {
        <svg viewBox="0 0 24 24">
          <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
          <path d="M14 2v6h6" />
        </svg>
      }
    }
  `,
  styles: `
    :host {
      display: inline-flex;
      width: 20px;
      height: 20px;
      color: inherit;
      flex-shrink: 0;
    }
    svg {
      width: 100%;
      height: 100%;
      fill: none;
      stroke: currentColor;
      stroke-width: 2;
      stroke-linecap: round;
      stroke-linejoin: round;
    }
  `,
})
export class TbIcon {
  readonly name = input.required<TbIconName>();
}
