import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Install } from '../../../src/components/Install';
import { AXIOS } from '../../../src/data';

describe('Install', () => {
  it('renders the Install heading and go get command', () => {
    const { container } = render(<Install lib={AXIOS} />);
    expect(container.querySelector(`#${AXIOS.id}-install`)).not.toBeNull();
    expect(screen.getByRole('heading', { name: 'Install' })).toBeInTheDocument();
    expect(screen.getByText(new RegExp(`go get ${AXIOS.pkg}`))).toBeInTheDocument();
  });
});
