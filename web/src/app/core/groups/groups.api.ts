import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';

export interface Group {
  id: string;
  name: string;
}

@Injectable({ providedIn: 'root' })
export class GroupsApi {
  private readonly http = inject(HttpClient);

  list() {
    return this.http.get<Group[]>('/groups');
  }
}
