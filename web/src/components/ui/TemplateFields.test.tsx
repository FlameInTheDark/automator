import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { TemplateInput, TemplateTextarea } from './TemplateFields'

describe('TemplateFields', () => {
  it('renders template tokens as layout-neutral marks with a trailing sentinel', () => {
    const { container } = render(
      <TemplateTextarea
        value={'hello {{input.data}}\n'}
        onChange={() => undefined}
        suggestions={[]}
      />,
    )

    const overlay = container.querySelector('[aria-hidden="true"]')
    const overlayContent = overlay?.firstElementChild as HTMLDivElement | null
    const token = overlayContent?.querySelector('mark')

    expect(token).not.toBeNull()
    expect(token).toHaveTextContent('{{input.data}}')
    expect(overlayContent?.querySelector('wbr')).not.toBeNull()
  })

  it('mirrors inline typography styles onto the highlight overlay', () => {
    const { container } = render(
      <TemplateInput
        value="{{input.data}}"
        onChange={() => undefined}
        suggestions={[]}
        style={{
          fontFamily: 'monospace',
          fontSize: '13px',
          lineHeight: '24px',
          paddingLeft: '20px',
        }}
      />,
    )

    const overlay = container.querySelector('[aria-hidden="true"]')
    const overlayContent = overlay?.firstElementChild as HTMLDivElement | null

    expect(overlayContent).toHaveStyle({
      fontFamily: 'monospace',
      fontSize: '13px',
      lineHeight: '24px',
      paddingLeft: '20px',
    })
  })

  it('keeps the overlay scroll position in sync with the textarea', () => {
    const longValue = Array.from({ length: 20 }, (_, index) => `line ${index} {{input.data}}`).join('\n')
    const { container } = render(
      <TemplateTextarea
        value={longValue}
        onChange={() => undefined}
        suggestions={[]}
        rows={2}
      />,
    )

    const textarea = screen.getByRole('textbox')
    const overlay = container.querySelector('[aria-hidden="true"]') as HTMLDivElement | null

    Object.defineProperty(textarea, 'scrollTop', {
      configurable: true,
      value: 96,
      writable: true,
    })
    Object.defineProperty(textarea, 'scrollLeft', {
      configurable: true,
      value: 14,
      writable: true,
    })

    fireEvent.scroll(textarea)

    expect(overlay?.scrollTop).toBe(96)
    expect(overlay?.scrollLeft).toBe(14)
  })
})
