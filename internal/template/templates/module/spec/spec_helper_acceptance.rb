# frozen_string_literal: true

require 'voxpupuli/acceptance/spec_helper_acceptance'

configure_beaker do |host|
  case fact('os.family')
  when 'Debian'
    install_puppet_module_via_pmt_on(host, 'puppetlabs-apt')
  when 'RedHat'
    if fact_on(host, 'os.name') == 'OracleLinux'
      ver = fact_on(host, 'os.release.major')
      install_package(host, "oracle-epel-release-el#{ver}")
    else
      # Soft dep on epel for Passenger
      install_package(host, 'epel-release')
    end
  end
end