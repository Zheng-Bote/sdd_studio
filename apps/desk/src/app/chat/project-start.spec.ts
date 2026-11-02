import { TestBed } from '@angular/core/testing';
import { ProjectStart } from './project-start';
import { ProjectState } from '../core/types';

function render(state: ProjectState | undefined) {
  const fixture = TestBed.createComponent(ProjectStart);
  fixture.componentRef.setInput('state', state);
  const started: string[] = [];
  fixture.componentInstance.start.subscribe((p) => started.push(p));
  fixture.detectChanges();
  return { el: fixture.nativeElement as HTMLElement, started };
}

describe('ProjectStart', () => {
  // desk-project-start AC-3: a folder with files is offered the take-over, an empty one a new project.
  it('offers the take-over for an existing folder', () => {
    const { el, started } = render('existing');
    expect(el.textContent).toContain('noch nicht mit sdd erfasst');
    const button = el.querySelector('button') as HTMLButtonElement;
    expect(button.textContent?.trim()).toBe('Bestehendes Projekt übernehmen');
    button.click();
    expect(started).toEqual(['sdd-adopt']);
  });

  it('offers a new project for an empty folder', () => {
    const { el, started } = render('empty');
    (el.querySelector('button') as HTMLButtonElement).click();
    expect(el.textContent).toContain('Neues Projekt anlegen');
    expect(started).toEqual(['sdd-start']);
  });

  it('shows nothing for a configured project or an unknown state', () => {
    expect(render('configured').el.textContent?.trim()).toBe('');
    expect(render(undefined).el.textContent?.trim()).toBe('');
  });
});
