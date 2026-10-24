import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { IOCList } from '../components/IOCList'

describe('IOCList', () => {
  it('renders nothing for an empty array', () => {
    const { container } = render(<IOCList iocs={[]} />)
    expect(container.firstChild).toBeNull()
  })

  it('renders AbuseIPDB link for an IPv4 address', () => {
    render(<IOCList iocs={['192.168.1.1']} />)
    const link = screen.getByRole('link')
    expect(link).toHaveAttribute('href', expect.stringContaining('abuseipdb.com/check/192.168.1.1'))
    expect(link).toHaveAttribute('target', '_blank')
  })

  it('renders VirusTotal link for an MD5 hash', () => {
    const md5 = 'd41d8cd98f00b204e9800998ecf8427e'
    render(<IOCList iocs={[md5]} />)
    const link = screen.getByRole('link')
    expect(link).toHaveAttribute('href', expect.stringContaining('virustotal.com'))
    expect(link).toHaveAttribute('href', expect.stringContaining(md5))
  })

  it('renders VirusTotal domain link for a domain', () => {
    render(<IOCList iocs={['evil.example.com']} />)
    const link = screen.getByRole('link')
    expect(link).toHaveAttribute('href', expect.stringContaining('virustotal.com'))
    expect(link).toHaveAttribute('href', expect.stringContaining('evil.example.com'))
  })

  it('renders multiple IOCs', () => {
    render(<IOCList iocs={['192.168.1.1', 'malware.exe', 'd41d8cd98f00b204e9800998ecf8427e']} />)
    // At least the IP and hash produce links; filename is unknown type (no link)
    const links = screen.getAllByRole('link')
    expect(links.length).toBeGreaterThanOrEqual(2)
  })

  it('renders IOC text inside the chip', () => {
    render(<IOCList iocs={['192.168.1.1']} />)
    expect(screen.getByText('192.168.1.1')).toBeInTheDocument()
  })
})
